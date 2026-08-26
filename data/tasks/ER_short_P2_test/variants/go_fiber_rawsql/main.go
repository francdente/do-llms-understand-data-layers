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
	database := os.Getenv("DB_PATH")
	if database == "" {
		database = "app.db"
	}

	var err error
	db, err = sql.Open("sqlite", database)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	app := fiber.New()

	app.Post("/api/products/:p_productkey/discontinue", discontinueProduct)
	app.Get("/api/products/:p_productkey/availability", getProductAvailability)

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	log.Fatal(app.Listen("0.0.0.0:" + port))
}

func discontinueProduct(c *fiber.Ctx) error {
	pProductkey, err := strconv.Atoi(c.Params("p_productkey"))
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Product not found"})
	}

	var existing int
	err = db.QueryRow(
		"SELECT p_productkey FROM product WHERE p_productkey = ?",
		pProductkey,
	).Scan(&existing)
	if err == sql.ErrNoRows {
		return c.Status(404).JSON(fiber.Map{"error": "Product not found"})
	}
	if err != nil {
		return err
	}

	_, err = db.Exec(
		"UPDATE product SET p_status = 'discontinued' WHERE p_productkey = ?",
		pProductkey,
	)
	if err != nil {
		return err
	}

	return c.Status(200).JSON(fiber.Map{
		"p_productkey": pProductkey,
		"p_status":     "discontinued",
	})
}

func getProductAvailability(c *fiber.Ctx) error {
	/**
	 * Get availability for purchasable product
	 */
	pProductkey, err := strconv.Atoi(c.Params("p_productkey"))
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Product not found"})
	}

	var row struct {
		PProductkey int     `json:"p_productkey"`
		PName       string  `json:"p_name"`
		TotalStock  float64 `json:"total_stock"`
	}

	err = db.QueryRow(
		"SELECT p.p_productkey, p.p_name, "+
			"       COALESCE(SUM(i.i_quantity), 0) AS total_stock "+
			"FROM product p LEFT JOIN inventory i ON p.p_productkey = i.i_productkey "+
			"WHERE p.p_productkey = ? "+
			"GROUP BY p.p_productkey",
		pProductkey,
	).Scan(&row.PProductkey, &row.PName, &row.TotalStock)
	if err == sql.ErrNoRows {
		return c.Status(404).JSON(fiber.Map{"error": "Product not found"})
	}
	if err != nil {
		return err
	}

	return c.Status(200).JSON(row)
}