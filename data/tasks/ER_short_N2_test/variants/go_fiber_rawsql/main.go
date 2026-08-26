package main

import (
	"database/sql"
	"log"
	"os"
	"strconv"

	"github.com/gofiber/fiber/v2"
	_ "modernc.org/sqlite"
)

type Product struct {
	PProductkey  int      `json:"p_productkey"`
	PName        *string  `json:"p_name"`
	PStatus      *string  `json:"p_status"`
	PRetailprice *float64 `json:"p_retailprice"`
}

type UpdatePricePayload struct {
	PRetailprice interface{} `json:"p_retailprice"`
}

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "app.db"
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	app := fiber.New()

	app.Put("/api/products/:p_productkey/price", func(c *fiber.Ctx) error {
		pProductkey, err := strconv.Atoi(c.Params("p_productkey"))
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Product not found"})
		}

		var payload map[string]interface{}
		if err := c.BodyParser(&payload); err != nil {
			payload = map[string]interface{}{}
		}
		if payload == nil {
			payload = map[string]interface{}{}
		}

		var existingKey int
		err = db.QueryRow(
			"SELECT p_productkey FROM product WHERE p_productkey = ?",
			pProductkey,
		).Scan(&existingKey)
		if err == sql.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Product not found"})
		}
		if err != nil {
			return err
		}

		_, err = db.Exec(
			"UPDATE product SET p_retailprice = ? WHERE p_productkey = ?",
			payload["p_retailprice"], pProductkey,
		)
		if err != nil {
			return err
		}

		var product Product
		err = db.QueryRow(
			"SELECT p_productkey, p_name, p_status, p_retailprice FROM product WHERE p_productkey = ?",
			pProductkey,
		).Scan(&product.PProductkey, &product.PName, &product.PStatus, &product.PRetailprice)
		if err != nil {
			return err
		}

		return c.Status(fiber.StatusOK).JSON(product)
	})

	app.Get("/api/products/:p_productkey", func(c *fiber.Ctx) error {
		pProductkey, err := strconv.Atoi(c.Params("p_productkey"))
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Product not found"})
		}

		var product Product
		err = db.QueryRow(
			"SELECT p_productkey, p_name, p_status, p_retailprice FROM product WHERE p_productkey = ?",
			pProductkey,
		).Scan(&product.PProductkey, &product.PName, &product.PStatus, &product.PRetailprice)
		if err == sql.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Product not found"})
		}
		if err != nil {
			return err
		}

		return c.Status(fiber.StatusOK).JSON(product)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	log.Fatal(app.Listen("0.0.0.0:" + port))
}