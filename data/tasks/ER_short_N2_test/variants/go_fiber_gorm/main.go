package main

import (
	"log"
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

type UpdatePricePayload struct {
	PRetailprice float64 `json:"p_retailprice"`
}

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "app.db"
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal(err)
	}
	defer sqlDB.Close()

	app := fiber.New()

	app.Put("/api/products/:p_productkey/price", func(c *fiber.Ctx) error {
		pProductkey, err := strconv.Atoi(c.Params("p_productkey"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid p_productkey")
		}

		var payload UpdatePricePayload
		if err := c.BodyParser(&payload); err != nil {
			payload = UpdatePricePayload{}
		}

		var existing struct {
			PProductkey int `gorm:"column:p_productkey"`
		}
		err = db.Table("product").
			Select("p_productkey").
			Where("p_productkey = ?", pProductkey).
			Take(&existing).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Product not found"})
			}
			return err
		}

		if err := db.Table("product").
			Where("p_productkey = ?", pProductkey).
			Update("p_retailprice", payload.PRetailprice).Error; err != nil {
			return err
		}

		var row Product
		err = db.Table("product").
			Select("p_productkey, p_name, p_status, p_retailprice").
			Where("p_productkey = ?", pProductkey).
			Take(&row).Error
		if err != nil {
			return err
		}

		return c.Status(fiber.StatusOK).JSON(row)
	})

	app.Get("/api/products/:p_productkey", func(c *fiber.Ctx) error {
		pProductkey, err := strconv.Atoi(c.Params("p_productkey"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid p_productkey")
		}

		var row Product
		err = db.Table("product").
			Select("p_productkey, p_name, p_status, p_retailprice").
			Where("p_productkey = ?", pProductkey).
			Take(&row).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Product not found"})
			}
			return err
		}

		return c.Status(fiber.StatusOK).JSON(row)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	log.Fatal(app.Listen("0.0.0.0:" + port))
}