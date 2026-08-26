package main

import (
	"database/sql"
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

	app.Get("/api/orders/:o_orderkey<int>", getOrder)
	app.Put("/api/orders/:o_orderkey<int>/lineitems/:l_linenumber<int>", updateLineitem)
	app.Post("/api/orders", createOrder)

	port := getEnv("PORT", "5000")
	_ = app.Listen("0.0.0.0:" + port)
}

func getOrder(c *fiber.Ctx) error {
	oOrderkey, err := strconv.Atoi(c.Params("o_orderkey"))
	if err != nil {
		return fiber.ErrBadRequest
	}

	db, err := getDB()
	if err != nil {
		return err
	}
	defer closeDB(db)

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

func updateLineitem(c *fiber.Ctx) error {
	oOrderkey, err := strconv.Atoi(c.Params("o_orderkey"))
	if err != nil {
		return fiber.ErrBadRequest
	}
	lLinenumber, err := strconv.Atoi(c.Params("l_linenumber"))
	if err != nil {
		return fiber.ErrBadRequest
	}

	payload := map[string]interface{}{}
	_ = c.BodyParser(&payload)

	db, err := getDB()
	if err != nil {
		return err
	}
	defer closeDB(db)

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	row := tx.QueryRow(
		"SELECT l_orderkey FROM lineitem WHERE l_orderkey = ? AND l_linenumber = ?",
		oOrderkey, lLinenumber,
	)

	var existing int
	err = row.Scan(&existing)
	if err == sql.ErrNoRows {
		_ = tx.Rollback()
		return c.Status(404).JSON(fiber.Map{"error": "Line item not found"})
	}
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	_, err = tx.Exec(
		"UPDATE lineitem SET l_extendedprice = ?, l_discount = ?, l_tax = ? "+
			"WHERE l_orderkey = ? AND l_linenumber = ?",
		payload["l_extendedprice"], payload["l_discount"], payload["l_tax"],
		oOrderkey, lLinenumber,
	)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	row = tx.QueryRow(
		"SELECT l_orderkey, l_linenumber, l_extendedprice, l_discount, l_tax "+
			"FROM lineitem WHERE l_orderkey = ? AND l_linenumber = ?",
		oOrderkey, lLinenumber,
	)

	var lOrderkey int
	var lLineNumber int
	var lExtendedprice float64
	var lDiscount float64
	var lTax float64

	err = row.Scan(&lOrderkey, &lLineNumber, &lExtendedprice, &lDiscount, &lTax)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return c.Status(200).JSON(fiber.Map{
		"l_orderkey":      lOrderkey,
		"l_linenumber":    lLineNumber,
		"l_extendedprice": lExtendedprice,
		"l_discount":      lDiscount,
		"l_tax":           lTax,
	})
}

func createOrder(c *fiber.Ctx) error {
	payload := map[string]interface{}{}
	_ = c.BodyParser(&payload)

	db, err := getDB()
	if err != nil {
		return err
	}
	defer closeDB(db)

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	oOrderstatus := "O"
	if v, ok := payload["o_orderstatus"]; ok && v != nil {
		if s, ok := v.(string); ok {
			oOrderstatus = s
		}
	}

	oComment := ""
	if v, ok := payload["o_comment"]; ok && v != nil {
		if s, ok := v.(string); ok {
			oComment = s
		}
	}

	res, err := tx.Exec(
		"INSERT INTO orders (o_orderstatus, o_totalprice, o_orderdate, o_comment) "+
			"VALUES (?, 0, ?, ?)",
		oOrderstatus, payload["o_orderdate"], oComment,
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
		var lShipdate interface{}
		if v, ok := item["l_shipdate"]; ok {
			lShipdate = v
		} else {
			lShipdate = nil
		}

		lComment := ""
		if v, ok := item["l_comment"]; ok && v != nil {
			if s, ok := v.(string); ok {
				lComment = s
			}
		}

		_, err = tx.Exec(
			"INSERT INTO lineitem (l_orderkey, l_linenumber, l_extendedprice, "+
				"l_discount, l_tax, l_shipdate, l_comment) "+
				"VALUES (?, ?, ?, ?, ?, ?, ?)",
			oOrderkey, item["l_linenumber"], item["l_extendedprice"],
			item["l_discount"], item["l_tax"],
			lShipdate, lComment,
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