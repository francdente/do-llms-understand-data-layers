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

type Customer struct {
	CCustkey int    `gorm:"column:c_custkey;primaryKey"`
	CName    string `gorm:"column:c_name"`
	CEmail   string `gorm:"column:c_email"`
}

func (Customer) TableName() string {
	return "customer"
}

type Order struct {
	OOrderkey   int     `gorm:"column:o_orderkey;primaryKey"`
	OCustkey    int     `gorm:"column:o_custkey"`
	OTotalprice float64 `gorm:"column:o_totalprice"`
	OOrderdate  string  `gorm:"column:o_orderdate"`
	OComment    string  `gorm:"column:o_comment"`
}

func (Order) TableName() string {
	return "orders"
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

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal(err)
	}
	defer sqlDB.Close()

	app.Delete("/api/customers/:c_custkey", func(c *fiber.Ctx) error {
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

	app.Get("/api/orders/summary", func(c *fiber.Ctx) error {
		/*
		   Return the summary of all existed historical orders
		*/
		type Summary struct {
			TotalOrders  int     `json:"total_orders"`
			TotalRevenue float64 `json:"total_revenue"`
		}

		var row Summary
		result := db.Raw(
			"SELECT COUNT(*) AS total_orders, " +
				"       COALESCE(SUM(o_totalprice), 0) AS total_revenue " +
				"FROM orders",
		).Scan(&row)

		if result.Error != nil {
			return result.Error
		}

		if row.TotalRevenue == 0 {
			var check sql.NullFloat64
			_ = check
		}

		return c.Status(200).JSON(row)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	log.Fatal(app.Listen("0.0.0.0:" + port))
}