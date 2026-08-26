package main

import (
	"database/sql"
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

type Inventory struct {
	IProductkey int     `gorm:"column:i_productkey;primaryKey"`
	IWarehouse  string  `gorm:"column:i_warehouse;primaryKey"`
	IQuantity   *int    `gorm:"column:i_quantity"`
}

func (Inventory) TableName() string {
	return "inventory"
}

type AvailabilityRow struct {
	PProductkey int            `json:"p_productkey" gorm:"column:p_productkey"`
	PName       sql.NullString `json:"-" gorm:"column:p_name"`
	TotalStock  int            `json:"total_stock" gorm:"column:total_stock"`
}

func main() {
	app := fiber.New()

	database := os.Getenv("DB_PATH")
	if database == "" {
		database = "app.db"
	}

	db, err := gorm.Open(sqlite.Open(database), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	app.Post("/api/products/:p_productkey/discontinue", func(c *fiber.Ctx) error {
		pProductkey, err := strconv.Atoi(c.Params("p_productkey"))
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "Product not found"})
		}

		var existing struct {
			PProductkey int `gorm:"column:p_productkey"`
		}
		result := db.Raw(
			"SELECT p_productkey FROM product WHERE p_productkey = ?",
			pProductkey,
		).Scan(&existing)

		if result.Error != nil {
			return c.Status(500).SendString(result.Error.Error())
		}
		if result.RowsAffected == 0 {
			return c.Status(404).JSON(fiber.Map{"error": "Product not found"})
		}

		result = db.Exec(
			"UPDATE product SET p_status = 'discontinued' WHERE p_productkey = ?",
			pProductkey,
		)
		if result.Error != nil {
			return c.Status(500).SendString(result.Error.Error())
		}

		return c.Status(200).JSON(fiber.Map{
			"p_productkey": pProductkey,
			"p_status":     "discontinued",
		})
	})

	app.Get("/api/products/:p_productkey/availability", func(c *fiber.Ctx) error {
		/**
		 * Get availability for purchasable product
		 */
		pProductkey, err := strconv.Atoi(c.Params("p_productkey"))
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "Product not found"})
		}

		var row AvailabilityRow
		result := db.Raw(
			"SELECT p.p_productkey, p.p_name, "+
				"       COALESCE(SUM(i.i_quantity), 0) AS total_stock "+
				"FROM product p LEFT JOIN inventory i ON p.p_productkey = i.i_productkey "+
				"WHERE p.p_productkey = ? "+
				"GROUP BY p.p_productkey",
			pProductkey,
		).Scan(&row)

		if result.Error != nil {
			return c.Status(500).SendString(result.Error.Error())
		}
		if result.RowsAffected == 0 {
			return c.Status(404).JSON(fiber.Map{"error": "Product not found"})
		}

		response := fiber.Map{
			"p_productkey": row.PProductkey,
			"p_name":       nil,
			"total_stock":  row.TotalStock,
		}
		if row.PName.Valid {
			response["p_name"] = row.PName.String
		}

		return c.Status(200).JSON(response)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	log.Fatal(app.Listen("0.0.0.0:" + port))
}