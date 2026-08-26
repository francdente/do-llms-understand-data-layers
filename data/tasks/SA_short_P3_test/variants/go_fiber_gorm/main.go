package main

import (
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Order struct {
	OOrderkey       int      `gorm:"column:o_orderkey;primaryKey" json:"o_orderkey"`
	OTotalprice     *float64 `gorm:"column:o_totalprice" json:"o_totalprice"`
	OOrderdate      *string  `gorm:"column:o_orderdate" json:"o_orderdate"`
	OLatestShipdate *string  `gorm:"column:o_latest_shipdate" json:"o_latest_shipdate"`
	OComment        *string  `gorm:"column:o_comment" json:"o_comment"`
}

func (Order) TableName() string {
	return "orders"
}

type Lineitem struct {
	LOrderkey      int      `gorm:"column:l_orderkey;primaryKey" json:"l_orderkey"`
	LLinenumber    int      `gorm:"column:l_linenumber;primaryKey" json:"l_linenumber"`
	LExtendedprice *float64 `gorm:"column:l_extendedprice" json:"l_extendedprice"`
	LDiscount      *float64 `gorm:"column:l_discount" json:"l_discount"`
	LTax           *float64 `gorm:"column:l_tax" json:"l_tax"`
	LShipdate      *string  `gorm:"column:l_shipdate" json:"l_shipdate"`
	LComment       *string  `gorm:"column:l_comment" json:"l_comment"`
}

func (Lineitem) TableName() string {
	return "lineitem"
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

	app.Post("/api/orders/:o_orderkey<int>/lineitems/:l_linenumber<int>/ship", func(c *fiber.Ctx) error {
		oOrderkey, err := c.ParamsInt("o_orderkey")
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid o_orderkey"})
		}
		lLinenumber, err := c.ParamsInt("l_linenumber")
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid l_linenumber"})
		}

		var existing struct {
			LOrderkey int `gorm:"column:l_orderkey"`
		}
		err = db.Table("lineitem").
			Select("l_orderkey").
			Where("l_orderkey = ? AND l_linenumber = ?", oOrderkey, lLinenumber).
			Take(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(404).JSON(fiber.Map{"error": "Line item not found"})
		}
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}

		today := time.Now().Format("2006-01-02")
		err = db.Table("lineitem").
			Where("l_orderkey = ? AND l_linenumber = ?", oOrderkey, lLinenumber).
			Update("l_shipdate", today).Error
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}

		return c.Status(200).JSON(fiber.Map{
			"l_orderkey":   oOrderkey,
			"l_linenumber": lLinenumber,
			"l_shipdate":   today,
		})
	})

	app.Get("/api/orders/:o_orderkey<int>", func(c *fiber.Ctx) error {
		oOrderkey, err := c.ParamsInt("o_orderkey")
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid o_orderkey"})
		}

		var row Order
		err = db.Table("orders").
			Select("o_orderkey, o_totalprice, o_orderdate, o_latest_shipdate, o_comment").
			Where("o_orderkey = ?", oOrderkey).
			Take(&row).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(404).JSON(fiber.Map{"error": "Order not found"})
		}
		if err != nil {
			return c.Status(500).SendString(err.Error())
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

	if err := app.Listen("0.0.0.0:" + port); err != nil {
		panic(err)
	}
}