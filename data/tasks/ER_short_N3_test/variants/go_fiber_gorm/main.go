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

type ArchivedProductResponse struct {
	PProductkey  int      `json:"p_productkey"`
	PName        *string  `json:"p_name"`
	PRetailprice *float64 `json:"p_retailprice"`
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

	app := fiber.New()

	app.Post("/api/products/:p_productkey/archive", func(c *fiber.Ctx) error {
		pProductkey, err := strconv.Atoi(c.Params("p_productkey"))
		if err != nil {
			return c.Status(fiber.StatusNotFound).SendString("Cannot POST /api/products/" + c.Params("p_productkey") + "/archive")
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
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Product not found"})
			}
			return result.Error
		}

		if err := db.Exec(
			"UPDATE product SET p_status = 'archived' WHERE p_productkey = ?",
			pProductkey,
		).Error; err != nil {
			return err
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"p_productkey": pProductkey,
			"p_status":     "archived",
		})
	})

	app.Get("/api/products/archived", func(c *fiber.Ctx) error {
		var rows []ArchivedProductResponse
		if err := db.Table("product").
			Select("p_productkey, p_name, p_retailprice").
			Where("p_status = ?", "archived").
			Order("p_productkey").
			Find(&rows).Error; err != nil {
			return err
		}
		return c.Status(fiber.StatusOK).JSON(rows)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	log.Fatal(app.Listen("0.0.0.0:" + port))
}