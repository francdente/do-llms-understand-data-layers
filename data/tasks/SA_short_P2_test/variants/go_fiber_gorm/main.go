package main

import (
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

type AddLineitemPayload struct {
	LLinenumber    int      `json:"l_linenumber"`
	LExtendedprice float64  `json:"l_extendedprice"`
	LDiscount      float64  `json:"l_discount"`
	LTax           float64  `json:"l_tax"`
	LShipdate      *string  `json:"l_shipdate"`
	LComment       *string  `json:"l_comment"`
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

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal(err)
	}
	defer sqlDB.Close()

	app := fiber.New()

	app.Get("/api/orders/:o_orderkey<int>", func(c *fiber.Ctx) error {
		oOrderkey, err := strconv.Atoi(c.Params("o_orderkey"))
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "Order not found"})
		}

		var row Order
		result := db.Select("o_orderkey, o_orderstatus, o_totalprice, o_orderdate, o_comment").
			Where("o_orderkey = ?", oOrderkey).
			First(&row)

		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return c.Status(404).JSON(fiber.Map{"error": "Order not found"})
		}
		if result.Error != nil {
			return result.Error
		}

		return c.Status(200).JSON(fiber.Map{
			"o_orderkey":    row.OOrderkey,
			"o_orderstatus": row.OOrderstatus,
			"o_totalprice":  row.OTotalprice,
			"o_orderdate":   row.OOrderdate,
			"o_comment":     row.OComment,
		})
	})

	app.Post("/api/orders/:o_orderkey<int>/lineitems", func(c *fiber.Ctx) error {
		payload := AddLineitemPayload{}
		if err := c.BodyParser(&payload); err != nil {
			payload = AddLineitemPayload{}
		}

		oOrderkey, err := strconv.Atoi(c.Params("o_orderkey"))
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "Order not found"})
		}

		tx := db.Begin()
		if tx.Error != nil {
			return tx.Error
		}

		var order struct {
			OOrderkey int `gorm:"column:o_orderkey"`
		}
		result := tx.Table("orders").Select("o_orderkey").Where("o_orderkey = ?", oOrderkey).Take(&order)
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			tx.Rollback()
			return c.Status(404).JSON(fiber.Map{"error": "Order not found"})
		}
		if result.Error != nil {
			tx.Rollback()
			return result.Error
		}

		comment := ""
		if payload.LComment != nil {
			comment = *payload.LComment
		}

		lineitem := Lineitem{
			LOrderkey:      oOrderkey,
			LLinenumber:    payload.LLinenumber,
			LExtendedprice: payload.LExtendedprice,
			LDiscount:      payload.LDiscount,
			LTax:           payload.LTax,
			LShipdate:      payload.LShipdate,
			LComment:       comment,
		}

		if err := tx.Table("lineitem").Create(&lineitem).Error; err != nil {
			tx.Rollback()
			return err
		}

		if err := tx.Commit().Error; err != nil {
			return err
		}

		return c.Status(201).JSON(fiber.Map{
			"l_orderkey":   oOrderkey,
			"l_linenumber": payload.LLinenumber,
		})
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

		orderStatus := "O"
		if payload.OOrderstatus != nil {
			orderStatus = *payload.OOrderstatus
		}

		comment := ""
		if payload.OComment != nil {
			comment = *payload.OComment
		}

		order := Order{
			OOrderstatus: orderStatus,
			OTotalprice:  0,
			OOrderdate:   payload.OOrderdate,
			OComment:     comment,
		}

		if err := tx.Table("orders").Create(&order).Error; err != nil {
			tx.Rollback()
			return err
		}

		oOrderkey := order.OOrderkey

		for _, item := range payload.Lineitems {
			itemComment := ""
			if item.LComment != nil {
				itemComment = *item.LComment
			}

			lineitem := Lineitem{
				LOrderkey:      oOrderkey,
				LLinenumber:    item.LLinenumber,
				LExtendedprice: item.LExtendedprice,
				LDiscount:      item.LDiscount,
				LTax:           item.LTax,
				LShipdate:      item.LShipdate,
				LComment:       itemComment,
			}

			if err := tx.Table("lineitem").Create(&lineitem).Error; err != nil {
				tx.Rollback()
				return err
			}
		}

		var totalRow struct {
			Total float64 `gorm:"column:total"`
		}
		if err := tx.Raw(
			"SELECT COALESCE(SUM(l_extendedprice * (1 - l_discount) * (1 + l_tax)), 0) AS total FROM lineitem WHERE l_orderkey = ?",
			oOrderkey,
		).Scan(&totalRow).Error; err != nil {
			tx.Rollback()
			return err
		}

		if err := tx.Table("orders").Where("o_orderkey = ?", oOrderkey).Update("o_totalprice", totalRow.Total).Error; err != nil {
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
	}

	log.Fatal(app.Listen("0.0.0.0:" + port))
}