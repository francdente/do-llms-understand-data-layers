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
	databasePath := os.Getenv("DB_PATH")
	if databasePath == "" {
		databasePath = "app.db"
	}

	var err error
	db, err = sql.Open("sqlite", databasePath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	app := fiber.New()

	app.Delete("/api/customers/:c_custkey", deleteCustomer)
	app.Get("/api/orders", listOrders)

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	log.Fatal(app.Listen("0.0.0.0:" + port))
}

func deleteCustomer(c *fiber.Ctx) error {
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
}

func listOrders(c *fiber.Ctx) error {
	/*
	   Return orders tied to active customers
	*/
	rows, err := db.Query(
		"SELECT o.o_orderkey, o.o_custkey, o.o_totalprice, o.o_orderdate, o.o_comment " +
			"FROM orders o JOIN customer c ON o.o_custkey = c.c_custkey " +
			"ORDER BY o.o_orderdate",
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	type Order struct {
		OOrderkey   int      `json:"o_orderkey"`
		OCustkey    int      `json:"o_custkey"`
		OTotalprice float64  `json:"o_totalprice"`
		OOrderdate  string   `json:"o_orderdate"`
		OComment    *string  `json:"o_comment"`
	}

	result := make([]Order, 0)
	for rows.Next() {
		var o Order
		if err := rows.Scan(&o.OOrderkey, &o.OCustkey, &o.OTotalprice, &o.OOrderdate, &o.OComment); err != nil {
			return err
		}
		result = append(result, o)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	return c.Status(200).JSON(result)
}