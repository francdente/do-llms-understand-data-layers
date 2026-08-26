package main

import (
	"database/sql"
	"log"
	"os"
	"strconv"

	"github.com/gofiber/fiber/v2"
	_ "modernc.org/sqlite"
)

var databasePath = getEnv("DB_PATH", "app.db")

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func openDB() (*sql.DB, error) {
	return sql.Open("sqlite", databasePath)
}

func main() {
	app := fiber.New()

	app.Post("/api/products/:p_productkey/discontinue", func(c *fiber.Ctx) error {
		pProductKey, err := strconv.Atoi(c.Params("p_productkey"))
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "Product not found"})
		}

		db, err := openDB()
		if err != nil {
			return err
		}
		defer db.Close()

		var existing int
		err = db.QueryRow(
			"SELECT p_productkey FROM product WHERE p_productkey = ?",
			pProductKey,
		).Scan(&existing)
		if err == sql.ErrNoRows {
			return c.Status(404).JSON(fiber.Map{"error": "Product not found"})
		}
		if err != nil {
			return err
		}

		_, err = db.Exec(
			"UPDATE product SET p_status = 'discontinued' WHERE p_productkey = ?",
			pProductKey,
		)
		if err != nil {
			return err
		}

		return c.Status(200).JSON(fiber.Map{
			"p_productkey": pProductKey,
			"p_status":     "discontinued",
		})
	})

	app.Get("/api/products", func(c *fiber.Ctx) error {
		/**
		 * List purchasable products
		 */
		db, err := openDB()
		if err != nil {
			return err
		}
		defer db.Close()

		rows, err := db.Query(
			"SELECT p_productkey, p_name, p_retailprice " +
				"FROM product WHERE p_status = 'active' " +
				"ORDER BY p_productkey",
		)
		if err != nil {
			return err
		}
		defer rows.Close()

		result := make([]fiber.Map, 0)
		for rows.Next() {
			var pProductKey int
			var pName sql.NullString
			var pRetailPrice sql.NullFloat64

			if err := rows.Scan(&pProductKey, &pName, &pRetailPrice); err != nil {
				return err
			}

			row := fiber.Map{
				"p_productkey": pProductKey,
				"p_name":       nil,
				"p_retailprice": nil,
			}
			if pName.Valid {
				row["p_name"] = pName.String
			}
			if pRetailPrice.Valid {
				row["p_retailprice"] = pRetailPrice.Float64
			}
			result = append(result, row)
		}
		if err := rows.Err(); err != nil {
			return err
		}

		return c.Status(200).JSON(result)
	})

	port := getEnv("PORT", "5000")
	log.Fatal(app.Listen("0.0.0.0:" + port))
}