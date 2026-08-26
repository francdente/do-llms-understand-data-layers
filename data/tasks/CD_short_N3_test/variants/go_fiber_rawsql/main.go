package main

import (
	"database/sql"
	"errors"
	"log"
	"os"
	"strconv"

	"github.com/gofiber/fiber/v2"
	_ "modernc.org/sqlite"
)

var db *sql.DB

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func initDB() *sql.DB {
	database := getEnv("DB_PATH", "app.db")
	conn, err := sql.Open("sqlite", database)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := conn.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		log.Fatal(err)
	}
	return conn
}

func main() {
	db = initDB()
	defer db.Close()

	app := fiber.New()

	app.Delete("/api/customers/:c_custkey", deleteCustomer)
	app.Get("/api/dashboard", dashboard)

	port := getEnv("PORT", "5000")
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
	if errors.Is(err, sql.ErrNoRows) {
		return c.Status(404).JSON(fiber.Map{"error": "Customer not found"})
	}
	if err != nil {
		return err
	}

	_, err = db.Exec("DELETE FROM customer WHERE c_custkey = ?", cCustkey)
	if err != nil {
		return c.Status(409).JSON(fiber.Map{"error": "Cannot delete customer with existing orders"})
	}

	return c.SendStatus(204)
}

func dashboard(c *fiber.Ctx) error {
	/**
	 * Return dashboard based on all existed orders
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
}