package main

import (
	"database/sql"
	"os"
	"strconv"

	"github.com/gofiber/fiber/v2"
	_ "modernc.org/sqlite"
)

func main() {
	database := os.Getenv("DB_PATH")
	if database == "" {
		database = "app.db"
	}

	db, err := sql.Open("sqlite", database)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	app := fiber.New()

	app.Delete("/api/customers/:c_custkey", func(c *fiber.Ctx) error {
		cCustkey, err := strconv.Atoi(c.Params("c_custkey"))
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "Customer not found"})
		}

		var existing int
		err = db.QueryRow(
			"SELECT c_custkey FROM customer WHERE c_custkey = ?",
			cCustkey,
		).Scan(&existing)
		if err == sql.ErrNoRows {
			return c.Status(404).JSON(fiber.Map{"error": "Customer not found"})
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

		return c.SendStatus(204)
	})

	app.Get("/api/orders/summary", func(c *fiber.Ctx) error {
		/*
		   Return the summary of all existed historical orders
		*/
		var totalOrders int
		var totalRevenue float64

		err := db.QueryRow(
			"SELECT COUNT(*) AS total_orders, " +
				"       COALESCE(SUM(o_totalprice), 0) AS total_revenue " +
				"FROM orders",
		).Scan(&totalOrders, &totalRevenue)
		if err != nil {
			return err
		}

		return c.Status(200).JSON(fiber.Map{
			"total_orders":  totalOrders,
			"total_revenue": totalRevenue,
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	if err := app.Listen("0.0.0.0:" + port); err != nil {
		panic(err)
	}
}