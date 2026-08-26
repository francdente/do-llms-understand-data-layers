package main

import (
	"fmt"
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

type OrderResponse struct {
	OOrderkey   int      `json:"o_orderkey"`
	OCustkey    *int     `json:"o_custkey"`
	OTotalprice *float64 `json:"o_totalprice"`
	OOrderdate  *string  `json:"o_orderdate"`
	OComment    *string  `json:"o_comment"`
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

	app.Delete("/api/customers/:c_custkey<int>", func(c *fiber.Ctx) error {
		cCustkey, err := strconv.Atoi(c.Params("c_custkey"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("")
		}

		var existing Customer
		result := db.Raw(
			"SELECT c_custkey FROM customer WHERE c_custkey = ?",
			cCustkey,
		).Scan(&existing)

		if result.Error != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("")
		}
		if result.RowsAffected == 0 {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Customer not found"})
		}

		tx := db.Begin()
		if tx.Error != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("")
		}

		if err := tx.Exec("DELETE FROM orders WHERE o_custkey = ?", cCustkey).Error; err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).SendString("")
		}
		if err := tx.Exec("DELETE FROM customer WHERE c_custkey = ?", cCustkey).Error; err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).SendString("")
		}
		if err := tx.Commit().Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("")
		}

		return c.SendStatus(fiber.StatusNoContent)
	})

	app.Get("/api/orders", func(c *fiber.Ctx) error {
		/**
		Return orders tied to active customers
		*/
		var rows []OrderResponse
		result := db.Raw(
			"SELECT o.o_orderkey, o.o_custkey, o.o_totalprice, o.o_orderdate, o.o_comment " +
				"FROM orders o JOIN customer c ON o.o_custkey = c.c_custkey " +
				"ORDER BY o.o_orderdate",
		).Scan(&rows)

		if result.Error != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("")
		}

		return c.Status(fiber.StatusOK).JSON(rows)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	if err := app.Listen(fmt.Sprintf("0.0.0.0:%s", port)); err != nil {
		panic(err)
	}
}