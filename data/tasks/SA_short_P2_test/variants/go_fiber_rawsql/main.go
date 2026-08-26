package main

import (
	"database/sql"
	"os"
	"strconv"

	"github.com/gofiber/fiber/v2"
	_ "modernc.org/sqlite"
)

var db *sql.DB

func getDB() *sql.DB {
	return db
}

func closeDB() {
	if db != nil {
		_ = db.Close()
	}
}

func getOrder(c *fiber.Ctx) error {
	oOrderkey, err := strconv.Atoi(c.Params("o_orderkey"))
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Order not found"})
	}

	db := getDB()
	row := db.QueryRow(
		"SELECT o_orderkey, o_orderstatus, o_totalprice, o_orderdate, o_comment "+
			"FROM orders WHERE o_orderkey = ?",
		oOrderkey,
	)

	var orderkey int
	var orderstatus string
	var totalprice float64
	var orderdate string
	var comment sql.NullString

	err = row.Scan(&orderkey, &orderstatus, &totalprice, &orderdate, &comment)
	if err == sql.ErrNoRows {
		return c.Status(404).JSON(fiber.Map{"error": "Order not found"})
	}
	if err != nil {
		return err
	}

	var commentVal interface{}
	if comment.Valid {
		commentVal = comment.String
	} else {
		commentVal = nil
	}

	return c.Status(200).JSON(fiber.Map{
		"o_orderkey":    orderkey,
		"o_orderstatus": orderstatus,
		"o_totalprice":  totalprice,
		"o_orderdate":   orderdate,
		"o_comment":     commentVal,
	})
}

func addLineitem(c *fiber.Ctx) error {
	oOrderkey, err := strconv.Atoi(c.Params("o_orderkey"))
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Order not found"})
	}

	payload := map[string]interface{}{}
	_ = c.BodyParser(&payload)

	db := getDB()
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	row := tx.QueryRow(
		"SELECT o_orderkey FROM orders WHERE o_orderkey = ?",
		oOrderkey,
	)
	var existingOrderkey int
	err = row.Scan(&existingOrderkey)
	if err == sql.ErrNoRows {
		_ = tx.Rollback()
		return c.Status(404).JSON(fiber.Map{"error": "Order not found"})
	}
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	_, err = tx.Exec(
		"INSERT INTO lineitem (l_orderkey, l_linenumber, l_extendedprice, l_discount, l_tax, l_shipdate, l_comment) "+
			"VALUES (?, ?, ?, ?, ?, ?, ?)",
		oOrderkey,
		payload["l_linenumber"],
		payload["l_extendedprice"],
		payload["l_discount"],
		payload["l_tax"],
		payload["l_shipdate"],
		func() interface{} {
			if v, ok := payload["l_comment"]; ok {
				return v
			}
			return ""
		}(),
	)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return c.Status(201).JSON(fiber.Map{
		"l_orderkey":   oOrderkey,
		"l_linenumber": payload["l_linenumber"],
	})
}

func createOrder(c *fiber.Ctx) error {
	payload := map[string]interface{}{}
	_ = c.BodyParser(&payload)

	db := getDB()
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	res, err := tx.Exec(
		"INSERT INTO orders (o_orderstatus, o_totalprice, o_orderdate, o_comment) "+
			"VALUES (?, 0, ?, ?)",
		func() interface{} {
			if v, ok := payload["o_orderstatus"]; ok {
				return v
			}
			return "O"
		}(),
		payload["o_orderdate"],
		func() interface{} {
			if v, ok := payload["o_comment"]; ok {
				return v
			}
			return ""
		}(),
	)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	oOrderkey64, err := res.LastInsertId()
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	oOrderkey := int(oOrderkey64)

	lineitemsRaw := payload["lineitems"].([]interface{})
	for _, itemRaw := range lineitemsRaw {
		item := itemRaw.(map[string]interface{})
		_, err = tx.Exec(
			"INSERT INTO lineitem (l_orderkey, l_linenumber, l_extendedprice, "+
				"l_discount, l_tax, l_shipdate, l_comment) "+
				"VALUES (?, ?, ?, ?, ?, ?, ?)",
			oOrderkey,
			item["l_linenumber"],
			item["l_extendedprice"],
			item["l_discount"],
			item["l_tax"],
			item["l_shipdate"],
			func() interface{} {
				if v, ok := item["l_comment"]; ok {
					return v
				}
				return ""
			}(),
		)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	row := tx.QueryRow(
		"SELECT COALESCE(SUM(l_extendedprice * (1 - l_discount) * (1 + l_tax)), 0) AS total "+
			"FROM lineitem WHERE l_orderkey = ?",
		oOrderkey,
	)
	var total float64
	if err := row.Scan(&total); err != nil {
		_ = tx.Rollback()
		return err
	}

	_, err = tx.Exec(
		"UPDATE orders SET o_totalprice = ? WHERE o_orderkey = ?",
		total, oOrderkey,
	)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return c.Status(201).JSON(fiber.Map{
		"o_orderkey":   oOrderkey,
		"o_totalprice": total,
	})
}

func main() {
	database := os.Getenv("DB_PATH")
	if database == "" {
		database = "app.db"
	}

	var err error
	db, err = sql.Open("sqlite", database)
	if err != nil {
		panic(err)
	}
	defer closeDB()

	app := fiber.New()

	app.Get("/api/orders/:o_orderkey<int>", getOrder)
	app.Post("/api/orders/:o_orderkey<int>/lineitems", addLineitem)
	app.Post("/api/orders", createOrder)

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	if err := app.Listen("0.0.0.0:" + port); err != nil {
		panic(err)
	}
}