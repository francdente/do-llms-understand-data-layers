package main

import (
	"log"
	"os"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Customer struct {
	CCustkey int     `gorm:"column:c_custkey;primaryKey" json:"c_custkey"`
	CName    *string `gorm:"column:c_name" json:"c_name"`
	CEmail   *string `gorm:"column:c_email" json:"c_email"`
}

func (Customer) TableName() string {
	return "customer"
}

type Order struct {
	OOrderkey   int      `gorm:"column:o_orderkey;primaryKey" json:"o_orderkey"`
	OCustkey    *int     `gorm:"column:o_custkey" json:"o_custkey"`
	OTotalprice *float64 `gorm:"column:o_totalprice" json:"o_totalprice"`
	OOrderdate  *string  `gorm:"column:o_orderdate" json:"o_orderdate"`
	OComment    *string  `gorm:"column:o_comment" json:"o_comment"`
}

func (Order) TableName() string {
	return "orders"
}

func main() {
	database := os.Getenv("DB_PATH")
	if database == "" {
		database = "app.db"
	}

	db, err := gorm.Open(sqlite.Open(database), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	app := fiber.New()

	app.Delete("/api/customers/:c_custkey<int>", func(c *fiber.Ctx) error {
		cCustkey, err := strconv.Atoi(c.Params("c_custkey"))
		if err != nil {
			return c.Status(404).SendString("")
		}

		var existing Customer
		result := db.Raw(
			"SELECT c_custkey FROM customer WHERE c_custkey = ?",
			cCustkey,
		).Scan(&existing)

		if result.Error != nil {
			return c.Status(500).SendString(result.Error.Error())
		}
		if result.RowsAffected == 0 {
			return c.Status(404).JSON(fiber.Map{"error": "Customer not found"})
		}

		tx := db.Begin()
		if tx.Error != nil {
			return c.Status(500).SendString(tx.Error.Error())
		}

		if err := tx.Exec("DELETE FROM orders WHERE o_custkey = ?", cCustkey).Error; err != nil {
			tx.Rollback()
			return c.Status(500).SendString(err.Error())
		}
		if err := tx.Exec("DELETE FROM customer WHERE c_custkey = ?", cCustkey).Error; err != nil {
			tx.Rollback()
			return c.Status(500).SendString(err.Error())
		}
		if err := tx.Commit().Error; err != nil {
			return c.Status(500).SendString(err.Error())
		}

		return c.SendStatus(204)
	})

	app.Get("/api/orders", func(c *fiber.Ctx) error {
		/*
		   Return orders tied to active customers
		*/
		var rows []Order
		result := db.Raw(
			"SELECT o_orderkey, o_custkey, o_totalprice, o_orderdate, o_comment " +
				"FROM orders WHERE o_custkey IN (SELECT c_custkey FROM customer) " +
				"ORDER BY o_orderdate",
		).Scan(&rows)

		if result.Error != nil {
			return c.Status(500).SendString(result.Error.Error())
		}

		return c.Status(200).JSON(rows)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	log.Fatal(app.Listen("0.0.0.0:" + port))
}