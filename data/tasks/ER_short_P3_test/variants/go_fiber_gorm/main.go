package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Supplier struct {
	SSuppkey int    `gorm:"column:s_suppkey;primaryKey" json:"s_suppkey"`
	SName    string `gorm:"column:s_name" json:"s_name"`
	SStatus  string `gorm:"column:s_status" json:"s_status"`
}

func (Supplier) TableName() string {
	return "supplier"
}

type Product struct {
	PProductkey  int     `gorm:"column:p_productkey;primaryKey" json:"p_productkey"`
	PName        string  `gorm:"column:p_name" json:"p_name"`
	PSuppkey     int     `gorm:"column:p_suppkey" json:"p_suppkey"`
	PRetailprice float64 `gorm:"column:p_retailprice" json:"p_retailprice"`
}

func (Product) TableName() string {
	return "product"
}

type ProductListRow struct {
	PProductkey  int     `json:"p_productkey"`
	PName        string  `json:"p_name"`
	PRetailprice float64 `json:"p_retailprice"`
	SupplierName string  `json:"supplier_name"`
}

func main() {
	app := fiber.New()

	database := os.Getenv("DB_PATH")
	if database == "" {
		database = "app.db"
	}

	db, err := gorm.Open(sqlite.Open(database), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	app.Post("/api/suppliers/:s_suppkey<int>/suspend", func(c *fiber.Ctx) error {
		sSuppkey, err := strconv.Atoi(c.Params("s_suppkey"))
		if err != nil {
			return c.Status(400).SendString("Bad Request")
		}

		var existing struct {
			SSuppkey int `gorm:"column:s_suppkey"`
		}
		result := db.Table("supplier").
			Select("s_suppkey").
			Where("s_suppkey = ?", sSuppkey).
			Take(&existing)

		if result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				return c.Status(404).JSON(fiber.Map{"error": "Supplier not found"})
			}
			return c.Status(500).SendString("Internal Server Error")
		}

		if err := db.Table("supplier").
			Where("s_suppkey = ?", sSuppkey).
			Update("s_status", "suspended").Error; err != nil {
			return c.Status(500).SendString("Internal Server Error")
		}

		return c.Status(200).JSON(fiber.Map{
			"s_suppkey": sSuppkey,
			"s_status":  "suspended",
		})
	})

	app.Get("/api/products", func(c *fiber.Ctx) error {
		/*
		   List purchasable products
		*/
		var rows []ProductListRow
		err := db.Table("product p").
			Select("p.p_productkey, p.p_name, p.p_retailprice, s.s_name AS supplier_name").
			Joins("JOIN supplier s ON p.p_suppkey = s.s_suppkey").
			Order("p.p_productkey").
			Scan(&rows).Error
		if err != nil {
			return c.Status(500).SendString("Internal Server Error")
		}
		return c.Status(200).JSON(rows)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	if err := app.Listen(fmt.Sprintf("0.0.0.0:%s", port)); err != nil {
		panic(err)
	}
}