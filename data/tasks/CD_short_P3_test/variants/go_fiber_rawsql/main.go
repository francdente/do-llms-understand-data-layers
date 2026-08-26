package main

import (
	"database/sql"
	"log"
	"os"
	"strconv"

	"github.com/gofiber/fiber/v2"
	_ "modernc.org/sqlite"
)

var db *sql.DB

func main() {
	var err error
	databasePath := os.Getenv("DB_PATH")
	if databasePath == "" {
		databasePath = "app.db"
	}

	db, err = sql.Open("sqlite", databasePath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	app := fiber.New()

	app.Delete("/api/customers/:c_custkey", deleteCustomer)
	app.Get("/api/orders/monthly-trend", monthlyTrend)

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	log.Fatal(app.Listen("0.0.0.0:" + port))
}

func deleteCustomer(c *fiber.Ctx) error {
	cCustkey, err := strconv.Atoi(c.Params("c_custkey"))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Customer not found"})
	}

	var existing int
	err = db.QueryRow(
		"SELECT c_custkey FROM customer WHERE c_custkey = ?",
		cCustkey,
	).Scan(&existing)
	if err == sql.ErrNoRows {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Customer not found"})
	}
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM orders WHERE o_custkey = ?", cCustkey)
	if err != nil {
		return err
	}

	_, err = tx.Exec("DELETE FROM customer WHERE c_custkey = ?", cCustkey)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func monthlyTrend(c *fiber.Ctx) error {
	/*
	   Return the trend of all existed orders per-month.
	*/
	rows, err := db.Query(
		"SELECT strftime('%Y-%m', o_orderdate) AS month, " +
			"       COUNT(*) AS order_count, " +
			"       SUM(o_totalprice) AS revenue " +
			"FROM orders " +
			"GROUP BY month " +
			"ORDER BY month",
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	result := make([]map[string]interface{}, 0)
	for rows.Next() {
		var month string
		var orderCount int64
		var revenue sql.NullFloat64

		if err := rows.Scan(&month, &orderCount, &revenue); err != nil {
			return err
		}

		var revenueValue interface{}
		if revenue.Valid {
			revenueValue = revenue.Float64
		} else {
			revenueValue = nil
		}

		result = append(result, map[string]interface{}{
			"month":       month,
			"order_count": orderCount,
			"revenue":     revenueValue,
		})
	}

	if err := rows.Err(); err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(result)
}