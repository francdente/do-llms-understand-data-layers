package main

import (
	"database/sql"
	"log"
	"os"
	"strconv"

	"github.com/gofiber/fiber/v2"
	_ "modernc.org/sqlite"
)

type App struct {
	db *sql.DB
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

	appState := &App{db: db}

	app := fiber.New()

	app.Post("/api/suppliers/:s_suppkey/suspend", appState.suspendSupplier)
	app.Get("/api/products", appState.listProducts)

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	log.Fatal(app.Listen("0.0.0.0:" + port))
}

func (a *App) suspendSupplier(c *fiber.Ctx) error {
	sSuppkey, err := strconv.Atoi(c.Params("s_suppkey"))
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Supplier not found"})
	}

	var existing int
	err = a.db.QueryRow(
		"SELECT s_suppkey FROM supplier WHERE s_suppkey = ?",
		sSuppkey,
	).Scan(&existing)
	if err == sql.ErrNoRows {
		return c.Status(404).JSON(fiber.Map{"error": "Supplier not found"})
	}
	if err != nil {
		return err
	}

	_, err = a.db.Exec(
		"UPDATE supplier SET s_status = 'suspended' WHERE s_suppkey = ?",
		sSuppkey,
	)
	if err != nil {
		return err
	}

	return c.Status(200).JSON(fiber.Map{
		"s_suppkey": sSuppkey,
		"s_status":  "suspended",
	})
}

func (a *App) listProducts(c *fiber.Ctx) error {
	/**
	 * List purchasable products
	 */
	rows, err := a.db.Query(
		"SELECT p.p_productkey, p.p_name, p.p_retailprice, s.s_name AS supplier_name " +
			"FROM product p JOIN supplier s ON p.p_suppkey = s.s_suppkey " +
			"ORDER BY p.p_productkey",
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	result := make([]fiber.Map, 0)
	for rows.Next() {
		var pProductkey int
		var pName sql.NullString
		var pRetailprice sql.NullFloat64
		var supplierName sql.NullString

		if err := rows.Scan(&pProductkey, &pName, &pRetailprice, &supplierName); err != nil {
			return err
		}

		result = append(result, fiber.Map{
			"p_productkey":  pProductkey,
			"p_name":        nullStringValue(pName),
			"p_retailprice": nullFloat64Value(pRetailprice),
			"supplier_name": nullStringValue(supplierName),
		})
	}

	if err := rows.Err(); err != nil {
		return err
	}

	return c.Status(200).JSON(result)
}

func nullStringValue(ns sql.NullString) interface{} {
	if ns.Valid {
		return ns.String
	}
	return nil
}

func nullFloat64Value(nf sql.NullFloat64) interface{} {
	if nf.Valid {
		return nf.Float64
	}
	return nil
}