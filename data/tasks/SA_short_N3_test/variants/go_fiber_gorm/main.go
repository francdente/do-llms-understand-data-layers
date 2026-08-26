package main

import (
	"log"
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

var db *gorm.DB

func main() {
	database := os.Getenv("DB_PATH")
	if database == "" {
		database = "app.db"
	}

	var err error
	db, err = gorm.Open(sqlite.Open(database), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	app := fiber.New()

	app.Post("/api/orders/:o_orderkey<int>/lineitems/:l_linenumber<int>/ship", shipLineitem)
	app.Get("/api/orders/:o_orderkey<int>", getOrder)

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	log.Fatal(app.Listen("0.0.0.0:" + port))
}

func shipLineitem(c *fiber.Ctx) error {
	oOrderkey, err := strconv.Atoi(c.Params("o_orderkey"))
	if err != nil {
		return err
	}
	lLinenumber, err := strconv.Atoi(c.Params("l_linenumber"))
	if err != nil {
		return err
	}

	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	var existing struct {
		LOrderkey int `gorm:"column:l_orderkey"`
	}
	err = tx.Raw(
		"SELECT l_orderkey FROM lineitem WHERE l_orderkey = ? AND l_linenumber = ?",
		oOrderkey, lLinenumber,
	).Scan(&existing).Error
	if err != nil {
		tx.Rollback()
		return err
	}
	if existing.LOrderkey == 0 {
		var count int64
		err = tx.Raw(
			"SELECT l_orderkey FROM lineitem WHERE l_orderkey = ? AND l_linenumber = ?",
			oOrderkey, lLinenumber,
		).RowsAffected
		_ = count
	}
	var existsCheck Lineitem
	res := tx.Raw(
		"SELECT l_orderkey, l_linenumber, l_extendedprice, l_discount, l_tax, l_shipdate, l_comment FROM lineitem WHERE l_orderkey = ? AND l_linenumber = ?",
		oOrderkey, lLinenumber,
	).Scan(&existsCheck)
	if res.Error != nil {
		tx.Rollback()
		return res.Error
	}
	if res.RowsAffected == 0 {
		tx.Rollback()
		return c.Status(404).JSON(fiber.Map{"error": "Line item not found"})
	}

	today := time.Now().Format("2006-01-02")

	if err := tx.Exec(
		"UPDATE lineitem SET l_shipdate = ? WHERE l_orderkey = ? AND l_linenumber = ?",
		today, oOrderkey, lLinenumber,
	).Error; err != nil {
		tx.Rollback()
		return err
	}

	var latestRow struct {
		Latest *string `gorm:"column:latest"`
	}
	if err := tx.Raw(
		"SELECT MAX(l_shipdate) AS latest FROM lineitem WHERE l_orderkey = ?",
		oOrderkey,
	).Scan(&latestRow).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Exec(
		"UPDATE orders SET o_latest_shipdate = ? WHERE o_orderkey = ?",
		latestRow.Latest, oOrderkey,
	).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
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
		return err
	}

	type OrderRow struct {
		OOrderkey       int      `gorm:"column:o_orderkey" json:"o_orderkey"`
		OTotalprice     *float64 `gorm:"column:o_totalprice" json:"o_totalprice"`
		OOrderdate      *string  `gorm:"column:o_orderdate" json:"o_orderdate"`
		OLatestShipdate *string  `gorm:"column:o_latest_shipdate" json:"o_latest_shipdate"`
		OComment        *string  `gorm:"column:o_comment" json:"o_comment"`
	}

	var row OrderRow
	res := db.Raw(
		"SELECT o_orderkey, o_totalprice, o_orderdate, o_latest_shipdate, o_comment FROM orders WHERE o_orderkey = ?",
		oOrderkey,
	).Scan(&row)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return c.Status(404).JSON(fiber.Map{"error": "Order not found"})
	}

	return c.Status(200).JSON(row)
}