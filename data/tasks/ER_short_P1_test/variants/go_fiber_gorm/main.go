package main

import (
	"os"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Product struct {
	PProductkey  int      `gorm:"column:p_productkey;primaryKey" json:"p_productkey"`
	PName        *string  `gorm:"column:p_name" json:"p_name"`
	PStatus      *string  `gorm:"column:p_status" json:"p_status"`
	PRetailprice *float64 `gorm:"column:p_retailprice" json:"p_retailprice"`
}

func (Product) TableName() string {
	return "product"
}

func main() {
	database := os.Getenv("DB_PATH")
	if database == "" {
		database = "app.db"
	}

	db, err := gorm.Open(sqlite.Open(database), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	app := fiber.New()

	app.Post("/api/products/:p_productkey/discontinue", func(c *fiber.Ctx) error {
		pProductkey, err := strconv.Atoi(c.Params("p_productkey"))
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "Product not found"})
		}

		var existing struct {
			PProductkey int `gorm:"column:p_productkey"`
		}
		result := db.Table("product").
			Select("p_productkey").
			Where("p_productkey = ?", pProductkey).
			Take(&existing)

		if result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				return c.Status(404).JSON(fiber.Map{"error": "Product not found"})
			}
			return result.Error
		}

		if err := db.Table("product").
			Where("p_productkey = ?", pProductkey).
			Update("p_status", "discontinued").Error; err != nil {
			return err
		}

		return c.Status(200).JSON(fiber.Map{
			"p_productkey": pProductkey,
			"p_status":     "discontinued",
		})
	})

	app.Get("/api/products/:p_productkey", func(c *fiber.Ctx) error {
		/*
		   Return purchasable product
		*/
		pProductkey, err := strconv.Atoi(c.Params("p_productkey"))
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "Product not found"})
		}

		var row Product
		result := db.Table("product").
			Select("p_productkey, p_name, p_status, p_retailprice").
			Where("p_productkey = ?", pProductkey).
			Take(&row)

		if result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				return c.Status(404).JSON(fiber.Map{"error": "Product not found"})
			}
			return result.Error
		}

		return c.Status(200).JSON(row)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	if err := app.Listen("0.0.0.0:" + port); err != nil {
		panic(err)
	}
}