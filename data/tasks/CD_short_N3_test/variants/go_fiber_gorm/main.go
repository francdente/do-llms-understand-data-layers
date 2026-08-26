package main

import (
	"errors"
	"os"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Customer struct {
	CCustkey int    `gorm:"column:c_custkey;primaryKey" json:"c_custkey"`
	CName    string `gorm:"column:c_name" json:"c_name"`
	CEmail   string `gorm:"column:c_email" json:"c_email"`
}

func (Customer) TableName() string {
	return "customer"
}

type Order struct {
	OOrderkey   int     `gorm:"column:o_orderkey;primaryKey" json:"o_orderkey"`
	OCustkey    int     `gorm:"column:o_custkey" json:"o_custkey"`
	OTotalprice float64 `gorm:"column:o_totalprice" json:"o_totalprice"`
	OOrderdate  string  `gorm:"column:o_orderdate" json:"o_orderdate"`
	OComment    string  `gorm:"column:o_comment" json:"o_comment"`
}

func (Order) TableName() string {
	return "orders"
}

func main() {
	database := os.Getenv("DB_PATH")
	if database == "" {
		database = "app.db"
	}

	db, err := gorm.Open(sqlite.Open(database), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		panic(err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		panic(err)
	}
	defer sqlDB.Close()

	if err := db.Exec("PRAGMA foreign_keys = ON;").Error; err != nil {
		panic(err)
	}

	app := fiber.New()

	app.Delete("/api/customers/:c_custkey", func(c *fiber.Ctx) error {
		cCustkey, err := strconv.Atoi(c.Params("c_custkey"))
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "Customer not found"})
		}

		var existing struct {
			CCustkey int `gorm:"column:c_custkey"`
		}
		err = db.Raw(
			"SELECT c_custkey FROM customer WHERE c_custkey = ?",
			cCustkey,
		).Scan(&existing).Error
		if err != nil {
			return err
		}
		if existing.CCustkey == 0 {
			return c.Status(404).JSON(fiber.Map{"error": "Customer not found"})
		}

		err = db.Exec("DELETE FROM customer WHERE c_custkey = ?", cCustkey).Error
		if err != nil {
			if errors.Is(err, sqlite3ConstraintErrorSentinel()) || isSQLiteConstraintError(err) {
				return c.Status(409).JSON(fiber.Map{"error": "Cannot delete customer with existing orders"})
			}
			return err
		}

		return c.SendStatus(204)
	})

	app.Get("/api/dashboard", func(c *fiber.Ctx) error {
		/*
		Return dashboard based on all existed orders
		*/
		var row struct {
			TotalOrders  int     `json:"total_orders"`
			TotalRevenue float64 `json:"total_revenue"`
		}

		err := db.Raw(
			"SELECT COUNT(*) AS total_orders, " +
				"       COALESCE(SUM(o_totalprice), 0) AS total_revenue " +
				"FROM orders",
		).Scan(&row).Error
		if err != nil {
			return err
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

func sqlite3ConstraintErrorSentinel() error {
	return nil
}

func isSQLiteConstraintError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return contains(msg, "FOREIGN KEY constraint failed") || contains(msg, "constraint failed")
}

func contains(s, sub string) bool {
	return len(sub) == 0 || indexOf(s, sub) >= 0
}

func indexOf(s, sub string) int {
	n := len(s)
	m := len(sub)
	if m == 0 {
		return 0
	}
	if m > n {
		return -1
	}
	for i := 0; i <= n-m; i++ {
		match := true
		for j := 0; j < m; j++ {
			if s[i+j] != sub[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}