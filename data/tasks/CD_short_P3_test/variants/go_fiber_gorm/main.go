package main

import (
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

type MonthlyTrendRow struct {
	Month      *string  `json:"month"`
	OrderCount int64    `json:"order_count"`
	Revenue    *float64 `json:"revenue"`
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

	app.Delete("/api/customers/:c_custkey<int>", func(c *fiber.Ctx) error {
		cCustkey, err := strconv.Atoi(c.Params("c_custkey"))
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "Customer not found"})
		}

		var existing struct {
			CCustkey int `gorm:"column:c_custkey"`
		}
		result := db.Raw(
			"SELECT c_custkey FROM customer WHERE c_custkey = ?",
			cCustkey,
		).Scan(&existing)

		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return c.Status(404).JSON(fiber.Map{"error": "Customer not found"})
		}

		tx := db.Begin()
		if tx.Error != nil {
			return tx.Error
		}

		if err := tx.Exec("DELETE FROM orders WHERE o_custkey = ?", cCustkey).Error; err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Exec("DELETE FROM customer WHERE c_custkey = ?", cCustkey).Error; err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit().Error; err != nil {
			return err
		}

		return c.SendStatus(204)
	})

	app.Get("/api/orders/monthly-trend", func(c *fiber.Ctx) error {
		/*
		   Return the trend of all existed orders per-month.
		*/
		var rows []MonthlyTrendRow
		result := db.Raw(
			"SELECT strftime('%Y-%m', o_orderdate) AS month, " +
				"       COUNT(*) AS order_count, " +
				"       SUM(o_totalprice) AS revenue " +
				"FROM orders " +
				"GROUP BY month " +
				"ORDER BY month",
		).Scan(&rows)
		if result.Error != nil {
			return result.Error
		}
		return c.Status(200).JSON(rows)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	if err := app.Listen("0.0.0.0:" + port); err != nil {
		panic(err)
	}
}