package main

import (
	"database/sql"
	"os"
	"strconv"
	"time"

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

func main() {
	app := fiber.New()

	app.Post("/api/orders/:o_orderkey/lineitems/:l_linenumber/ship", shipLineitem)
	app.Get("/api/orders/:o_orderkey", getOrder)

	port := getEnv("PORT", "5000")
	app.Listen("0.0.0.0:" + port)
}

func openDB() (*sql.DB, error) {
	return sql.Open("sqlite", DATABASE)
}

func shipLineitem(c *fiber.Ctx) error {
	oOrderkey, err := strconv.Atoi(c.Params("o_orderkey"))
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Line item not found"})
	}
	lLinenumber, err := strconv.Atoi(c.Params("l_linenumber"))
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Line item not found"})
	}

	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	var existing int
	err = db.QueryRow(
		"SELECT l_orderkey FROM lineitem WHERE l_orderkey = ? AND l_linenumber = ?",
		oOrderkey, lLinenumber,
	).Scan(&existing)
	if err == sql.ErrNoRows {
		return c.Status(404).JSON(fiber.Map{"error": "Line item not found"})
	}
	if err != nil {
		return err
	}

	today := time.Now().Format("2006-01-02")
	_, err = db.Exec(
		"UPDATE lineitem SET l_shipdate = ? WHERE l_orderkey = ? AND l_linenumber = ?",
		today, oOrderkey, lLinenumber,
	)
	if err != nil {
		return err
	}

	return c.Status(200).JSON(fiber.Map{
		"l_orderkey":   oOrderkey,
		"l_linenumber": lLinenumber,
		"l_shipdate":   today,
	})
}

func getOrder(c *fiber.Ctx) error {
	oOrderkey, err := strconv.Atoi(c.Params("o_orderkey"))
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Order not found"})
	}

	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	var oTotalprice sql.NullFloat64
	var oOrderdate sql.NullString
	var oLatestShipdate sql.NullString
	var oComment sql.NullString

	err = db.QueryRow(
		"SELECT o_orderkey, o_totalprice, o_orderdate, o_latest_shipdate, o_comment FROM orders WHERE o_orderkey = ?",
		oOrderkey,
	).Scan(&oOrderkey, &oTotalprice, &oOrderdate, &oLatestShipdate, &oComment)
	if err == sql.ErrNoRows {
		return c.Status(404).JSON(fiber.Map{"error": "Order not found"})
	}
	if err != nil {
		return err
	}

	resp := fiber.Map{
		"o_orderkey": oOrderkey,
	}

	if oTotalprice.Valid {
		resp["o_totalprice"] = oTotalprice.Float64
	} else {
		resp["o_totalprice"] = nil
	}
	if oOrderdate.Valid {
		resp["o_orderdate"] = oOrderdate.String
	} else {
		resp["o_orderdate"] = nil
	}
	if oLatestShipdate.Valid {
		resp["o_latest_shipdate"] = oLatestShipdate.String
	} else {
		resp["o_latest_shipdate"] = nil
	}
	if oComment.Valid {
		resp["o_comment"] = oComment.String
	} else {
		resp["o_comment"] = nil
	}

	return c.Status(200).JSON(resp)
}