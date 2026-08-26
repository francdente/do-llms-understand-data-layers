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

func main() {
	app := fiber.New()

	app.Post("/api/products/:p_productkey/discontinue", func(c *fiber.Ctx) error {
		pProductkey, err := strconv.Atoi(c.Params("p_productkey"))
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "Product not found"})
		}

		db, err := getDB()
		if err != nil {
			log.Printf("database open error: %v", err)
			return c.SendStatus(500)
		}
		defer db.Close()

		var existing int
		err = db.QueryRow(
			"SELECT p_productkey FROM product WHERE p_productkey = ?",
			pProductkey,
		).Scan(&existing)
		if err == sql.ErrNoRows {
			return c.Status(404).JSON(fiber.Map{"error": "Product not found"})
		}
		if err != nil {
			log.Printf("query error: %v", err)
			return c.SendStatus(500)
		}

		_, err = db.Exec(
			"UPDATE product SET p_status = 'discontinued' WHERE p_productkey = ?",
			pProductkey,
		)
		if err != nil {
			log.Printf("update error: %v", err)
			return c.SendStatus(500)
		}

		return c.Status(200).JSON(fiber.Map{
			"p_productkey": pProductkey,
			"p_status":     "discontinued",
		})
	})

	app.Get("/api/products/:p_productkey", func(c *fiber.Ctx) error {
		/**
		Return purchasable product
		*/
		pProductkey, err := strconv.Atoi(c.Params("p_productkey"))
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "Product not found"})
		}

		db, err := getDB()
		if err != nil {
			log.Printf("database open error: %v", err)
			return c.SendStatus(500)
		}
		defer db.Close()

		var pProductkeyOut int
		var pName sql.NullString
		var pStatus sql.NullString
		var pRetailprice sql.NullFloat64

		err = db.QueryRow(
			"SELECT p_productkey, p_name, p_status, p_retailprice FROM product WHERE p_productkey = ?",
			pProductkey,
		).Scan(&pProductkeyOut, &pName, &pStatus, &pRetailprice)
		if err == sql.ErrNoRows {
			return c.Status(404).JSON(fiber.Map{"error": "Product not found"})
		}
		if err != nil {
			log.Printf("query error: %v", err)
			return c.SendStatus(500)
		}

		return c.Status(200).JSON(fiber.Map{
			"p_productkey":  pProductkeyOut,
			"p_name":        nullStringToAny(pName),
			"p_status":      nullStringToAny(pStatus),
			"p_retailprice": nullFloat64ToAny(pRetailprice),
		})
	})

	port := getEnv("PORT", "5000")
	log.Fatal(app.Listen("0.0.0.0:" + port))
}

func nullStringToAny(ns sql.NullString) interface{} {
	if ns.Valid {
		return ns.String
	}
	return nil
}

func nullFloat64ToAny(nf sql.NullFloat64) interface{} {
	if nf.Valid {
		return nf.Float64
	}
	return nil
}