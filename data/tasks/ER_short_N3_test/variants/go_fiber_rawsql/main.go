package main

import (
	"database/sql"
	"log"
	"os"
	"strconv"

	"github.com/gofiber/fiber/v2"
	_ "modernc.org/sqlite"
)

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

	app.Post("/api/products/:p_productkey/archive", func(c *fiber.Ctx) error {
		pProductKeyStr := c.Params("p_productkey")
		pProductKey, err := strconv.Atoi(pProductKeyStr)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Product not found"})
		}

		var existing int
		err = db.QueryRow(
			"SELECT p_productkey FROM product WHERE p_productkey = ?",
			pProductKey,
		).Scan(&existing)
		if err == sql.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Product not found"})
		}
		if err != nil {
			return err
		}

		_, err = db.Exec(
			"UPDATE product SET p_status = 'archived' WHERE p_productkey = ?",
			pProductKey,
		)
		if err != nil {
			return err
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"p_productkey": pProductKey,
			"p_status":     "archived",
		})
	})

	app.Get("/api/products/archived", func(c *fiber.Ctx) error {
		rows, err := db.Query(
			"SELECT p_productkey, p_name, p_retailprice " +
				"FROM product WHERE p_status = 'archived' " +
				"ORDER BY p_productkey",
		)
		if err != nil {
			return err
		}
		defer rows.Close()

		result := make([]map[string]interface{}, 0)
		for rows.Next() {
			var pProductKey int
			var pName sql.NullString
			var pRetailPrice sql.NullFloat64

			if err := rows.Scan(&pProductKey, &pName, &pRetailPrice); err != nil {
				return err
			}

			row := map[string]interface{}{
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

		return c.Status(fiber.StatusOK).JSON(result)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	log.Fatal(app.Listen("0.0.0.0:" + port))
}