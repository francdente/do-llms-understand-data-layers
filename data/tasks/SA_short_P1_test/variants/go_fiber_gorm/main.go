package main

import (
	"database/sql"
	"errors"
	"os"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Order struct {
	OOrderkey    int     `gorm:"column:o_orderkey;primaryKey;autoIncrement" json:"o_orderkey"`
	OOrderstatus string  `gorm:"column:o_orderstatus" json:"o_orderstatus"`
	OTotalprice  float64 `gorm:"column:o_totalprice" json:"o_totalprice"`
	OOrderdate   string  `gorm:"column:o_orderdate" json:"o_orderdate"`
	OComment     string  `gorm:"column:o_comment" json:"o_comment"`
}

func (Order) TableName() string {
	return "orders"
}

type Lineitem struct {
	LOrderkey      int      `gorm:"column:l_orderkey;primaryKey" json:"l_orderkey"`
	LLinenumber    int      `gorm:"column:l_linenumber;primaryKey" json:"l_linenumber"`
	LExtendedprice float64  `gorm:"column:l_extendedprice" json:"l_extendedprice"`
	LDiscount      float64  `gorm:"column:l_discount" json:"l_discount"`
	LTax           float64  `gorm:"column:l_tax" json:"l_tax"`
	LShipdate      *string  `gorm:"column:l_shipdate" json:"l_shipdate"`
	LComment       string   `gorm:"column:l_comment" json:"l_comment"`
}

func (Lineitem) TableName() string {
	return "lineitem"
}

type UpdateLineitemPayload struct {
	LExtendedprice float64 `json:"l_extendedprice"`
	LDiscount      float64 `json:"l_discount"`
	LTax           float64 `json:"l_tax"`
}

type CreateOrderLineitemPayload struct {
	LLinenumber    int      `json:"l_linenumber"`
	LExtendedprice float64  `json:"l_extendedprice"`
	LDiscount      float64  `json:"l_discount"`
	LTax           float64  `json:"l_tax"`
	LShipdate      *string  `json:"l_shipdate"`
	LComment       *string  `json:"l_comment"`
}

type CreateOrderPayload struct {
	OOrderstatus *string                      `json:"o_orderstatus"`
	OOrderdate   string                       `json:"o_orderdate"`
	OComment     *string                      `json:"o_comment"`
	Lineitems    []CreateOrderLineitemPayload `json:"lineitems"`
}

func main() {
	database := os.Getenv("DB_PATH")
	if database == "" {
		database = "app.db"
	}

	db, err := gorm.Open(sqlite.Open(database), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	app := fiber.New()

	app.Get("/api/orders/:o_orderkey<int>", func(c *fiber.Ctx) error {
		oOrderkey, err := strconv.Atoi(c.Params("o_orderkey"))
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "Order not found"})
		}

		var row struct {
			OOrderkey    int     `json:"o_orderkey"`
			OOrderstatus string  `json:"o_orderstatus"`
			OTotalprice  float64 `json:"o_totalprice"`
			OOrderdate   string  `json:"o_orderdate"`
			OComment     string  `json:"o_comment"`
		}

		result := db.Raw(
			"SELECT o_orderkey, o_orderstatus, o_totalprice, o_orderdate, o_comment FROM orders WHERE o_orderkey = ?",
			oOrderkey,
		).Scan(&row)

		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return c.Status(404).JSON(fiber.Map{"error": "Order not found"})
		}
		return c.Status(200).JSON(row)
	})

	app.Put("/api/orders/:o_orderkey<int>/lineitems/:l_linenumber<int>", func(c *fiber.Ctx) error {
		oOrderkey, err := strconv.Atoi(c.Params("o_orderkey"))
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "Line item not found"})
		}
		lLinenumber, err := strconv.Atoi(c.Params("l_linenumber"))
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "Line item not found"})
		}

		payload := UpdateLineitemPayload{}
		if err := c.BodyParser(&payload); err != nil {
			payload = UpdateLineitemPayload{}
		}

		tx := db.Begin()
		if tx.Error != nil {
			return tx.Error
		}

		var existing struct {
			LOrderkey int `json:"l_orderkey"`
		}
		result := tx.Raw(
			"SELECT l_orderkey FROM lineitem WHERE l_orderkey = ? AND l_linenumber = ?",
			oOrderkey, lLinenumber,
		).Scan(&existing)
		if result.Error != nil {
			tx.Rollback()
			return result.Error
		}
		if result.RowsAffected == 0 {
			tx.Rollback()
			return c.Status(404).JSON(fiber.Map{"error": "Line item not found"})
		}

		if err := tx.Exec(
			"UPDATE lineitem SET l_extendedprice = ?, l_discount = ?, l_tax = ? WHERE l_orderkey = ? AND l_linenumber = ?",
			payload.LExtendedprice, payload.LDiscount, payload.LTax, oOrderkey, lLinenumber,
		).Error; err != nil {
			tx.Rollback()
			return err
		}

		var row struct {
			LOrderkey      int     `json:"l_orderkey"`
			LLinenumber    int     `json:"l_linenumber"`
			LExtendedprice float64 `json:"l_extendedprice"`
			LDiscount      float64 `json:"l_discount"`
			LTax           float64 `json:"l_tax"`
		}
		result = tx.Raw(
			"SELECT l_orderkey, l_linenumber, l_extendedprice, l_discount, l_tax FROM lineitem WHERE l_orderkey = ? AND l_linenumber = ?",
			oOrderkey, lLinenumber,
		).Scan(&row)
		if result.Error != nil {
			tx.Rollback()
			return result.Error
		}

		if err := tx.Commit().Error; err != nil {
			return err
		}
		return c.Status(200).JSON(row)
	})

	app.Post("/api/orders", func(c *fiber.Ctx) error {
		payload := CreateOrderPayload{}
		if err := c.BodyParser(&payload); err != nil {
			payload = CreateOrderPayload{}
		}

		tx := db.Begin()
		if tx.Error != nil {
			return tx.Error
		}

		oOrderstatus := "O"
		if payload.OOrderstatus != nil {
			oOrderstatus = *payload.OOrderstatus
		}
		oComment := ""
		if payload.OComment != nil {
			oComment = *payload.OComment
		}

		result := tx.Exec(
			"INSERT INTO orders (o_orderstatus, o_totalprice, o_orderdate, o_comment) VALUES (?, 0, ?, ?)",
			oOrderstatus, payload.OOrderdate, oComment,
		)
		if result.Error != nil {
			tx.Rollback()
			return result.Error
		}

		var oOrderkey int64
		row := tx.Raw("SELECT last_insert_rowid()").Row()
		if err := row.Scan(&oOrderkey); err != nil {
			tx.Rollback()
			return err
		}

		for _, item := range payload.Lineitems {
			lComment := ""
			if item.LComment != nil {
				lComment = *item.LComment
			}
			if err := tx.Exec(
				"INSERT INTO lineitem (l_orderkey, l_linenumber, l_extendedprice, l_discount, l_tax, l_shipdate, l_comment) VALUES (?, ?, ?, ?, ?, ?, ?)",
				oOrderkey, item.LLinenumber, item.LExtendedprice, item.LDiscount, item.LTax, item.LShipdate, lComment,
			).Error; err != nil {
				tx.Rollback()
				return err
			}
		}

		var totalRow struct {
			Total sql.NullFloat64 `json:"total"`
		}
		result = tx.Raw(
			"SELECT COALESCE(SUM(l_extendedprice * (1 - l_discount) * (1 + l_tax)), 0) AS total FROM lineitem WHERE l_orderkey = ?",
			oOrderkey,
		).Scan(&totalRow)
		if result.Error != nil {
			tx.Rollback()
			return result.Error
		}

		total := 0.0
		if totalRow.Total.Valid {
			total = totalRow.Total.Float64
		}

		if err := tx.Exec(
			"UPDATE orders SET o_totalprice = ? WHERE o_orderkey = ?",
			total, oOrderkey,
		).Error; err != nil {
			tx.Rollback()
			return err
		}

		if err := tx.Commit().Error; err != nil {
			return err
		}

		return c.Status(201).JSON(fiber.Map{
			"o_orderkey":   oOrderkey,
			"o_totalprice": total,
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	if err := app.Listen("0.0.0.0:" + port); err != nil && !errors.Is(err, fiber.ErrServerClosed) {
		panic(err)
	}
}