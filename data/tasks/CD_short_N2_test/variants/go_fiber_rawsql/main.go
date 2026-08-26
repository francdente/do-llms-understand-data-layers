package main

import (
	"database/sql"
	"log"
	"os"
	"strconv"

	"github.com/gofiber/fiber/v2"
	_ "modernc.org/sqlite"
)

type Order struct {
	OOrderkey   int      `json:"o_orderkey"`
	OCustkey    int      `json:"o_custkey"`
	OTotalprice *float64 `json:"o_totalprice"`
	OOrderdate  *string  `json:"o_orderdate"`
	OComment    *string  `json:"o_comment"`
}

func main() {
	database := os.Getenv("DB_PATH")
	if database == "" {
		database = "app.db"
	}

	db, err := sql.Open("sqlite", database)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	app := fiber.New()

	app.Delete("/api/customers/:c_custkey<int>", func(c *fiber.Ctx) error {
		cCustkey, err := strconv.Atoi(c.Params("c_custkey"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("")
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
	})

	app.Get("/api/orders", func(c *fiber.Ctx) error {
		/*
		   Return orders tied to active customers
		*/
		rows, err := db.Query(
			"SELECT o_orderkey, o_custkey, o_totalprice, o_orderdate, o_comment " +
				"FROM orders WHERE o_custkey IN (SELECT c_custkey FROM customer) " +
				"ORDER BY o_orderdate",
		)
		if err != nil {
			return err
		}
		defer rows.Close()

		orders := make([]Order, 0)
		for rows.Next() {
			var o Order
			if err := rows.Scan(&o.OOrderkey, &o.OCustkey, &o.OTotalprice, &o.OOrderdate, &o.OComment); err != nil {
				return err
			}
			orders = append(orders, o)
		}
		if err := rows.Err(); err != nil {
			return err
		}

		return c.Status(fiber.StatusOK).JSON(orders)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	log.Fatal(app.Listen("0.0.0.0:" + port))
}