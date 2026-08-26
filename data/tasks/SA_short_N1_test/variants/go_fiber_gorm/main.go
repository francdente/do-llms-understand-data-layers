package main

import (
	"database/sql"
	"errors"
	"log"
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
		log.Fatal(err)
	}

	app := fiber.New()

	app.Get("/api/orders/:o_orderkey<int>", func(c *fiber.Ctx) error {
		oOrderkey, err := c.ParamsInt("o_orderkey")
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

		res := db.Raw(
			"SELECT o_orderkey, o_orderstatus, o_totalprice, o_orderdate, o_comment FROM orders WHERE o_orderkey = ?",
			oOrderkey,
		).Scan(&row)

		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return c.Status(404).JSON(fiber.Map{"error": "Order not found"})
		}
		return c.Status(200).JSON(row)
	})

	app.Put("/api/orders/:o_orderkey<int>/lineitems/:l_linenumber<int>", func(c *fiber.Ctx) error {
		oOrderkey, err := c.ParamsInt("o_orderkey")
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "Line item not found"})
		}
		lLinenumber, err := c.ParamsInt("l_linenumber")
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "Line item not found"})
		}

		payload := UpdateLineitemPayload{}
		if body := c.Body(); len(body) > 0 {
			_ = c.BodyParser(&payload)
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
			LOrderkey int
		}
		res := tx.Raw(
			"SELECT l_orderkey FROM lineitem WHERE l_orderkey = ? AND l_linenumber = ?",
			oOrderkey, lLinenumber,
		).Scan(&existing)
		if res.Error != nil {
			tx.Rollback()
			return res.Error
		}
		if res.RowsAffected == 0 {
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

		if err := tx.Exec(
			"UPDATE orders SET o_totalprice = (SELECT COALESCE(SUM(l_extendedprice * (1 - l_discount) * (1 + l_tax)), 0) FROM lineitem WHERE l_orderkey = ?) WHERE o_orderkey = ?",
			oOrderkey, oOrderkey,
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
		res = tx.Raw(
			"SELECT l_orderkey, l_linenumber, l_extendedprice, l_discount, l_tax FROM lineitem WHERE l_orderkey = ? AND l_linenumber = ?",
			oOrderkey, lLinenumber,
		).Scan(&row)
		if res.Error != nil {
			tx.Rollback()
			return res.Error
		}

		if err := tx.Commit().Error; err != nil {
			return err
		}
		return c.Status(200).JSON(row)
	})

	app.Post("/api/orders", func(c *fiber.Ctx) error {
		payload := CreateOrderPayload{}
		if body := c.Body(); len(body) > 0 {
			_ = c.BodyParser(&payload)
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

		oOrderstatus := "O"
		if payload.OOrderstatus != nil {
			oOrderstatus = *payload.OOrderstatus
		}
		oComment := ""
		if payload.OComment != nil {
			oComment = *payload.OComment
		}

		res := tx.Exec(
			"INSERT INTO orders (o_orderstatus, o_totalprice, o_orderdate, o_comment) VALUES (?, 0, ?, ?)",
			oOrderstatus, payload.OOrderdate, oComment,
		)
		if res.Error != nil {
			tx.Rollback()
			return res.Error
		}

		var oOrderkey int64
		if res.Statement != nil && res.Statement.ConnPool != nil {
			if sqlDB, ok := res.Statement.ConnPool.(*sql.DB); ok && sqlDB != nil {
				_ = sqlDB
			}
		}
		oOrderkey = res.Statement.ConnPool.(gorm.ConnPool).(*sql.DB).Stats().MaxOpenConnections
		if oOrderkey == 0 {
			var id int64
			if err := tx.Raw("SELECT last_insert_rowid()").Scan(&id).Error; err != nil {
				tx.Rollback()
				return err
			}
			oOrderkey = id
		}

		if oOrderkey == 0 {
			var id int64
			if err := tx.Raw("SELECT last_insert_rowid()").Scan(&id).Error; err != nil {
				tx.Rollback()
				return err
			}
			oOrderkey = id
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
			Total float64 `json:"total"`
		}
		if err := tx.Raw(
			"SELECT COALESCE(SUM(l_extendedprice * (1 - l_discount) * (1 + l_tax)), 0) AS total FROM lineitem WHERE l_orderkey = ?",
			oOrderkey,
		).Scan(&totalRow).Error; err != nil {
			tx.Rollback()
			return err
		}

		if err := tx.Exec(
			"UPDATE orders SET o_totalprice = ? WHERE o_orderkey = ?",
			totalRow.Total, oOrderkey,
		).Error; err != nil {
			tx.Rollback()
			return err
		}

		if err := tx.Commit().Error; err != nil {
			return err
		}

		return c.Status(201).JSON(fiber.Map{
			"o_orderkey":   oOrderkey,
			"o_totalprice": totalRow.Total,
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	} else {
		if _, err := strconv.Atoi(port); err != nil {
			port = "5000"
		}
	}

	if err := app.Listen("0.0.0.0:" + port); err != nil && !errors.Is(err, fiber.ErrServerClosed) {
		log.Fatal(err)
	}
}