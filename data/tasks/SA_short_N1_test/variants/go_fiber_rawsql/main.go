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

func main() {
	db, err := sql.Open("sqlite", DATABASE)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	app := fiber.New()

	app.Get("/api/orders/:o_orderkey", func(c *fiber.Ctx) error {
		oOrderkey, err := strconv.Atoi(c.Params("o_orderkey"))
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "Order not found"})
		}

		row := db.QueryRow(
			"SELECT o_orderkey, o_orderstatus, o_totalprice, o_orderdate, o_comment "+
				"FROM orders WHERE o_orderkey = ?",
			oOrderkey,
		)

		var o_orderkey int
		var o_orderstatus string
		var o_totalprice float64
		var o_orderdate string
		var o_comment string

		err = row.Scan(&o_orderkey, &o_orderstatus, &o_totalprice, &o_orderdate, &o_comment)
		if err == sql.ErrNoRows {
			return c.Status(404).JSON(fiber.Map{"error": "Order not found"})
		}
		if err != nil {
			return err
		}

		return c.Status(200).JSON(fiber.Map{
			"o_orderkey":    o_orderkey,
			"o_orderstatus": o_orderstatus,
			"o_totalprice":  o_totalprice,
			"o_orderdate":   o_orderdate,
			"o_comment":     o_comment,
		})
	})

	app.Put("/api/orders/:o_orderkey/lineitems/:l_linenumber", func(c *fiber.Ctx) error {
		oOrderkey, err := strconv.Atoi(c.Params("o_orderkey"))
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "Line item not found"})
		}
		lLinenumber, err := strconv.Atoi(c.Params("l_linenumber"))
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "Line item not found"})
		}

		payload := map[string]interface{}{}
		if len(c.Body()) > 0 {
			if err := c.BodyParser(&payload); err != nil {
				payload = map[string]interface{}{}
			}
		}

		tx, err := db.Begin()
		if err != nil {
			return err
		}

		var existing int
		err = tx.QueryRow(
			"SELECT l_orderkey FROM lineitem WHERE l_orderkey = ? AND l_linenumber = ?",
			oOrderkey, lLinenumber,
		).Scan(&existing)
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

		_, err = tx.Exec(
			"UPDATE orders SET o_totalprice = "+
				"(SELECT COALESCE(SUM(l_extendedprice * (1 - l_discount) * (1 + l_tax)), 0) "+
				" FROM lineitem WHERE l_orderkey = ?) "+
				"WHERE o_orderkey = ?",
			oOrderkey, oOrderkey,
		)
		if err != nil {
			_ = tx.Rollback()
			return err
		}

		var l_orderkey int
		var l_linenumber int
		var l_extendedprice float64
		var l_discount float64
		var l_tax float64

		err = tx.QueryRow(
			"SELECT l_orderkey, l_linenumber, l_extendedprice, l_discount, l_tax "+
				"FROM lineitem WHERE l_orderkey = ? AND l_linenumber = ?",
			oOrderkey, lLinenumber,
		).Scan(&l_orderkey, &l_linenumber, &l_extendedprice, &l_discount, &l_tax)
		if err != nil {
			_ = tx.Rollback()
			return err
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		return c.Status(200).JSON(fiber.Map{
			"l_orderkey":      l_orderkey,
			"l_linenumber":    l_linenumber,
			"l_extendedprice": l_extendedprice,
			"l_discount":      l_discount,
			"l_tax":           l_tax,
		})
	})

	app.Post("/api/orders", func(c *fiber.Ctx) error {
		payload := map[string]interface{}{}
		if len(c.Body()) > 0 {
			if err := c.BodyParser(&payload); err != nil {
				payload = map[string]interface{}{}
			}
		}

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

		lastID, err := res.LastInsertId()
		if err != nil {
			_ = tx.Rollback()
			return err
		}
		oOrderkey := int(lastID)

		lineitemsRaw := payload["lineitems"].([]interface{})
		for _, itemRaw := range lineitemsRaw {
			item := itemRaw.(map[string]interface{})

			lShipdate, hasShipdate := item["l_shipdate"]
			if !hasShipdate {
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

		var total float64
		err = tx.QueryRow(
			"SELECT COALESCE(SUM(l_extendedprice * (1 - l_discount) * (1 + l_tax)), 0) AS total "+
				"FROM lineitem WHERE l_orderkey = ?",
			oOrderkey,
		).Scan(&total)
		if err != nil {
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
	})

	port := getEnv("PORT", "5000")
	log.Fatal(app.Listen("0.0.0.0:" + port))
}