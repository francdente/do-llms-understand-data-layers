package main

import (
	"errors"
	"os"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Order struct {
	OOrderkey    int    `gorm:"column:o_orderkey;primaryKey" json:"o_orderkey"`
	OOrderstatus string `gorm:"column:o_orderstatus" json:"o_orderstatus"`
	OOrderdate   string `gorm:"column:o_orderdate" json:"o_orderdate"`
	OComment     string `gorm:"column:o_comment" json:"o_comment"`
}

func (Order) TableName() string {
	return "orders"
}

type Lineitem struct {
	LOrderkey      int     `gorm:"column:l_orderkey;primaryKey" json:"l_orderkey"`
	LLinenumber    int     `gorm:"column:l_linenumber;primaryKey" json:"l_linenumber"`
	LExtendedprice float64 `gorm:"column:l_extendedprice" json:"l_extendedprice"`
	LDiscount      float64 `gorm:"column:l_discount" json:"l_discount"`
	LTax           float64 `gorm:"column:l_tax" json:"l_tax"`
	LShipdate      string  `gorm:"column:l_shipdate" json:"l_shipdate"`
	LComment       string  `gorm:"column:l_comment" json:"l_comment"`
}

func (Lineitem) TableName() string {
	return "lineitem"
}

type OrderResponse struct {
	OOrderkey    int     `json:"o_orderkey"`
	OOrderstatus string  `json:"o_orderstatus"`
	OOrderdate   string  `json:"o_orderdate"`
	OComment     string  `json:"o_comment"`
	OTotalprice  float64 `json:"o_totalprice"`
}

type UpdateLineitemPayload struct {
	LExtendedprice float64 `json:"l_extendedprice"`
	LDiscount      float64 `json:"l_discount"`
	LTax           float64 `json:"l_tax"`
}

type LineitemResponse struct {
	LOrderkey      int     `json:"l_orderkey"`
	LLinenumber    int     `json:"l_linenumber"`
	LExtendedprice float64 `json:"l_extendedprice"`
	LDiscount      float64 `json:"l_discount"`
	LTax           float64 `json:"l_tax"`
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
		oOrderkey, err := c.ParamsInt("o_orderkey")
		if err != nil {
			return c.Status(400).SendString("Invalid order key")
		}

		tx := db.Begin()
		if tx.Error != nil {
			return c.Status(500).JSON(fiber.Map{"error": tx.Error.Error()})
		}

		var order Order
		err = tx.Raw(
			"SELECT o_orderkey, o_orderstatus, o_orderdate, o_comment FROM orders WHERE o_orderkey = ?",
			oOrderkey,
		).Scan(&order).Error
		if err != nil {
			tx.Rollback()
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		if order.OOrderkey == 0 {
			tx.Rollback()
			return c.Status(404).JSON(fiber.Map{"error": "Order not found"})
		}

		var totalRow struct {
			Total float64 `gorm:"column:total"`
		}
		err = tx.Raw(
			"SELECT COALESCE(SUM(l_extendedprice * (1 - l_discount) * (1 + l_tax)), 0) AS total FROM lineitem WHERE l_orderkey = ?",
			oOrderkey,
		).Scan(&totalRow).Error
		if err != nil {
			tx.Rollback()
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		if err := tx.Commit().Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		return c.Status(200).JSON(OrderResponse{
			OOrderkey:    order.OOrderkey,
			OOrderstatus: order.OOrderstatus,
			OOrderdate:   order.OOrderdate,
			OComment:     order.OComment,
			OTotalprice:  totalRow.Total,
		})
	})

	app.Put("/api/orders/:o_orderkey<int>/lineitems/:l_linenumber<int>", func(c *fiber.Ctx) error {
		oOrderkey, err := c.ParamsInt("o_orderkey")
		if err != nil {
			return c.Status(400).SendString("Invalid order key")
		}
		lLinenumber, err := c.ParamsInt("l_linenumber")
		if err != nil {
			return c.Status(400).SendString("Invalid line number")
		}

		var payload map[string]interface{}
		if err := c.BodyParser(&payload); err != nil {
			payload = map[string]interface{}{}
		}

		lExtendedprice, ok := toFloat64(payload["l_extendedprice"])
		if !ok {
			return c.Status(500).JSON(fiber.Map{"error": "missing l_extendedprice"})
		}
		lDiscount, ok := toFloat64(payload["l_discount"])
		if !ok {
			return c.Status(500).JSON(fiber.Map{"error": "missing l_discount"})
		}
		lTax, ok := toFloat64(payload["l_tax"])
		if !ok {
			return c.Status(500).JSON(fiber.Map{"error": "missing l_tax"})
		}

		tx := db.Begin()
		if tx.Error != nil {
			return c.Status(500).JSON(fiber.Map{"error": tx.Error.Error()})
		}

		var existing struct {
			LOrderkey int `gorm:"column:l_orderkey"`
		}
		err = tx.Raw(
			"SELECT l_orderkey FROM lineitem WHERE l_orderkey = ? AND l_linenumber = ?",
			oOrderkey, lLinenumber,
		).Scan(&existing).Error
		if err != nil {
			tx.Rollback()
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		if existing.LOrderkey == 0 {
			tx.Rollback()
			return c.Status(404).JSON(fiber.Map{"error": "Line item not found"})
		}

		err = tx.Exec(
			"UPDATE lineitem SET l_extendedprice = ?, l_discount = ?, l_tax = ? WHERE l_orderkey = ? AND l_linenumber = ?",
			lExtendedprice, lDiscount, lTax, oOrderkey, lLinenumber,
		).Error
		if err != nil {
			tx.Rollback()
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		var row LineitemResponse
		err = tx.Raw(
			"SELECT l_orderkey, l_linenumber, l_extendedprice, l_discount, l_tax FROM lineitem WHERE l_orderkey = ? AND l_linenumber = ?",
			oOrderkey, lLinenumber,
		).Scan(&row).Error
		if err != nil {
			tx.Rollback()
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		if err := tx.Commit().Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		return c.Status(200).JSON(row)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}
	if _, err := strconv.Atoi(port); err != nil {
		port = "5000"
	}

	if err := app.Listen("0.0.0.0:" + port); err != nil && !errors.Is(err, fiber.ErrServerClosed) {
		panic(err)
	}
}

func toFloat64(v interface{}) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case float32:
		return float64(t), true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case int32:
		return float64(t), true
	case string:
		f, err := strconv.ParseFloat(t, 64)
		if err != nil {
			return 0, false
		}
		return f, true
	default:
		return 0, false
	}
}