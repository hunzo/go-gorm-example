package main

import (
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Users struct {
	gorm.Model
	Firstname string
	Lastname  string
	Attach    []Files `gorm:"foreignKey:Idx"`
}

type Files struct {
	Idx       int    `json:"idx"`
	FileBytes []byte `json:"file_bytes"`
}

type ReqUsers struct {
	FirstName string     `json:"first_name"`
	LastName  string     `json:"last_name"`
	Attach    []ReqFiles `json:"attachs,omitempty"`
}
type ReqFiles struct {
	FileId    int    `json:"file_id"`
	FileBytes string `json:"file_bytes"`
}

type UserRepository interface {
	GetUsers() ([]Users, error)
	GetUserById(int) (*Users, error)
	CreateUser(Users) error
}

//---------------------------------
type UserRepositoryDB struct {
	db *gorm.DB
}

func New(db *gorm.DB) UserRepositoryDB {
	db.AutoMigrate(&Users{}, &Files{})
	return UserRepositoryDB{
		db: db,
	}
}

func (r UserRepositoryDB) CreateUser(u Users) error {
	user := Users{
		Firstname: u.Firstname,
		Lastname:  u.Lastname,
		Attach:    u.Attach,
	}

	if err := r.db.Create(&user).Error; err != nil {
		return err
	}

	return nil
}

func (r UserRepositoryDB) DeleteUserById(id int) error {
	user := Users{}
	files := Files{}

	r.db.Unscoped().Delete(&user, "id = ?", id)
	r.db.Unscoped().Delete(&files, "file_id = ?", id)

	return nil
}

func (r UserRepositoryDB) GetUsers() ([]Users, error) {
	users := []Users{}
	r.db.Preload(clause.Associations).Find(&users)
	return users, nil
}
func (r UserRepositoryDB) GetUserById(id int) (*Users, error) {
	user := Users{}
	r.db.Where("id = ?", id).Preload(clause.Associations).Find(&user)
	fmt.Println(user)
	return &user, nil
}

func main() {
	app := fiber.New()
	app.Use(logger.New())
	app.Use(cors.New())

	db, err := gorm.Open(sqlite.Open("test.sqlite"), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	userRepo := New(db)

	app.Get("/", func(c *fiber.Ctx) error {
		u, err := userRepo.GetUsers()
		if err != nil {
			return c.SendStatus(fiber.StatusNotFound)
		}
		return c.JSON(u)
	})
	app.Get("/user/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")

		i, err := strconv.Atoi(id)
		if err != nil {
			return c.SendStatus(fiber.StatusBadRequest)
		}

		u, err := userRepo.GetUserById(i)
		if err != nil {
			return c.SendStatus(fiber.StatusNotFound)
		}
		return c.JSON(u)
	})

	app.Post("/create", func(c *fiber.Ctx) error {
		req := ReqUsers{}
		if err := c.BodyParser(&req); err != nil {
			return c.SendStatus(fiber.StatusBadRequest)
		}

		files := []Files{}

		fmt.Printf("Req: %v\n", req)

		for _, v := range req.Attach {
			// fmt.Printf("Att: K:%v V:%v", k, v)
			f := Files{
				FileBytes: []byte(v.FileBytes),
				Idx:       v.FileId,
			}
			files = append(files, f)
		}

		fmt.Printf("Files: %v\n", files)

		u := Users{
			Firstname: req.FirstName,
			Lastname:  req.LastName,
			Attach:    files,
		}

		fmt.Printf("Payload: %v\n", u)

		if err := userRepo.CreateUser(u); err != nil {
			return c.SendStatus(fiber.StatusUnprocessableEntity)
		}

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"sucess": true,
		})
	})

	app.Delete("/delete/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		if id == "" {
			return c.SendStatus(fiber.StatusBadRequest)
		}

		i, err := strconv.Atoi(id)
		if err != nil {
			return c.SendStatus(fiber.StatusBadRequest)
		}

		if err := userRepo.DeleteUserById(i); err != nil {
			return c.SendStatus(fiber.StatusNotFound)
		}

		return c.SendStatus(fiber.StatusNoContent)
	})

	app.Listen(":8080")

}
