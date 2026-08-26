package main

import (
	"database/sql"
	"log"
	"os"
	"strconv"

	"github.com/gofiber/fiber/v2"
	_ "modernc.org/sqlite"
)

var DATABASE = getEnv("DB_PATH", "app.db")

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite", DATABASE)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func closeDB(db *sql.DB) {
	if db != nil {
		_ = db.Close()
	}
}

func main() {
	app := fiber.New()

	app.Get("/api/orders/:o_orderkey<int>", func(c *fiber.Ctx) error {
		oOrderkey, err := strconv.Atoi(c.Params("o_orderkey"))
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid order key"})
		}

		db, err := getDB()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Database connection error"})
		}
		defer closeDB(db)

		tx, err := db.Begin()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Transaction error"})
		}

		var o_orderkey int
		var o_orderstatus, o_orderdate, o_comment sql.NullString

		row := tx.QueryRow(
			"SELECT o_orderkey, o_orderstatus, o_orderdate, o_comment "+
				"FROM orders WHERE o_orderkey = ?",
			oOrderkey,
		)
		err = row.Scan(&o_orderkey, &o_orderstatus, &o_orderdate, &o_comment)
		if err == sql.ErrNoRows {
			_ = tx.Rollback()
			return c.Status(404).JSON(fiber.Map{"error": "Order not found"})
		}
		if err != nil {
			_ = tx.Rollback()
			return c.Status(500).JSON(fiber.Map{"error": "Query error"})
		}

		var total float64
		row = tx.QueryRow(
			"SELECT COALESCE(SUM(l_extendedprice * (1 - l_discount) * (1 + l_tax)), 0) AS total "+
				"FROM lineitem WHERE l_orderkey = ?",
			oOrderkey,
		)
		if err := row.Scan(&total); err != nil {
			_ = tx.Rollback()
			return c.Status(500).JSON(fiber.Map{"error": "Query error"})
		}

		if err := tx.Commit(); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Commit error"})
		}

		return c.Status(200).JSON(fiber.Map{
			"o_orderkey":    o_orderkey,
			"o_orderstatus": nullableString(o_orderstatus),
			"o_orderdate":   nullableString(o_orderdate),
			"o_comment":     nullableString(o_comment),
			"o_totalprice":  total,
		})
	})

	app.Put("/api/orders/:o_orderkey<int>/lineitems/:l_linenumber<int>", func(c *fiber.Ctx) error {
		oOrderkey, err := strconv.Atoi(c.Params("o_orderkey"))
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid order key"})
		}
		lLinenumber, err := strconv.Atoi(c.Params("l_linenumber"))
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid line number"})
		}

		payload := map[string]interface{}{}
		if err := c.BodyParser(&payload); err != nil {
			payload = map[string]interface{}{}
		}

		db, err := getDB()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Database connection error"})
		}
		defer closeDB(db)

		tx, err := db.Begin()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Transaction error"})
		}

		var existingOrderkey int
		row := tx.QueryRow(
			"SELECT l_orderkey FROM lineitem WHERE l_orderkey = ? AND l_linenumber = ?",
			oOrderkey, lLinenumber,
		)
		err = row.Scan(&existingOrderkey)
		if err == sql.ErrNoRows {
			_ = tx.Rollback()
			return c.Status(404).JSON(fiber.Map{"error": "Line item not found"})
		}
		if err != nil {
			_ = tx.Rollback()
			return c.Status(500).JSON(fiber.Map{"error": "Query error"})
		}

		_, err = tx.Exec(
			"UPDATE lineitem SET l_extendedprice = ?, l_discount = ?, l_tax = ? "+
				"WHERE l_orderkey = ? AND l_linenumber = ?",
			payload["l_extendedprice"], payload["l_discount"], payload["l_tax"],
			oOrderkey, lLinenumber,
		)
		if err != nil {
			_ = tx.Rollback()
			return c.Status(500).JSON(fiber.Map{"error": "Update error"})
		}

		var l_orderkey, l_linenumber int
		var l_extendedprice, l_discount, l_tax float64
		row = tx.QueryRow(
			"SELECT l_orderkey, l_linenumber, l_extendedprice, l_discount, l_tax "+
				"FROM lineitem WHERE l_orderkey = ? AND l_linenumber = ?",
			oOrderkey, lLinenumber,
		)
		err = row.Scan(&l_orderkey, &l_linenumber, &l_extendedprice, &l_discount, &l_tax)
		if err != nil {
			_ = tx.Rollback()
			return c.Status(500).JSON(fiber.Map{"error": "Query error"})
		}

		if err := tx.Commit(); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Commit error"})
		}

		return c.Status(200).JSON(fiber.Map{
			"l_orderkey":      l_orderkey,
			"l_linenumber":    l_linenumber,
			"l_extendedprice": l_extendedprice,
			"l_discount":      l_discount,
			"l_tax":           l_tax,
		})
	})

	port := getEnv("PORT", "5000")
	log.Fatal(app.Listen("0.0.0.0:" + port))
}

func nullableString(ns sql.NullString) interface{} {
	if ns.Valid {
		return ns.String
	}
	return nil
}