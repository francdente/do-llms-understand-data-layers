package main

import (
	"database/sql"
	"errors"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Order struct {
	OOrderkey    int64   `gorm:"column:o_orderkey;primaryKey;autoIncrement" json:"o_orderkey"`
	OOrderstatus string  `gorm:"column:o_orderstatus" json:"o_orderstatus"`
	OTotalprice  float64 `gorm:"column:o_totalprice" json:"o_totalprice"`
	OOrderdate   string  `gorm:"column:o_orderdate" json:"o_orderdate"`
	OComment     string  `gorm:"column:o_comment" json:"o_comment"`
}

func (Order) TableName() string {
	return "orders"
}

type Lineitem struct {
	LOrderkey      int64    `gorm:"column:l_orderkey;primaryKey" json:"l_orderkey"`
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

type AddLineitemPayload struct {
	LLinenumber    int      `json:"l_linenumber"`
	LExtendedprice float64  `json:"l_extendedprice"`
	LDiscount      float64  `json:"l_discount"`
	LTax           float64  `json:"l_tax"`
	LShipdate      *string  `json:"l_shipdate"`
	LComment       *string  `json:"l_comment"`
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

	sqlDB, err := db.DB()
	if err != nil {
		panic(err)
	}
	defer sqlDB.Close()

	r := gin.Default()

	r.GET("/api/orders/:o_orderkey", func(c *gin.Context) {
		oOrderkey, err := strconv.ParseInt(c.Param("o_orderkey"), 10, 64)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
			return
		}

		var result struct {
			OOrderkey    int64   `json:"o_orderkey"`
			OOrderstatus string  `json:"o_orderstatus"`
			OTotalprice  float64 `json:"o_totalprice"`
			OOrderdate   string  `json:"o_orderdate"`
			OComment     string  `json:"o_comment"`
		}

		tx := db.Raw(
			"SELECT o_orderkey, o_orderstatus, o_totalprice, o_orderdate, o_comment FROM orders WHERE o_orderkey = ?",
			oOrderkey,
		).Scan(&result)

		if tx.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": tx.Error.Error()})
			return
		}
		if tx.RowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
			return
		}

		c.JSON(http.StatusOK, result)
	})

	r.POST("/api/orders/:o_orderkey/lineitems", func(c *gin.Context) {
		oOrderkey, err := strconv.ParseInt(c.Param("o_orderkey"), 10, 64)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
			return
		}

		var payload AddLineitemPayload
		if err := c.ShouldBindJSON(&payload); err != nil {
			payload = AddLineitemPayload{}
		}

		err = db.Transaction(func(tx *gorm.DB) error {
			var order struct {
				OOrderkey int64 `gorm:"column:o_orderkey"`
			}

			check := tx.Raw(
				"SELECT o_orderkey FROM orders WHERE o_orderkey = ?",
				oOrderkey,
			).Scan(&order)
			if check.Error != nil {
				return check.Error
			}
			if check.RowsAffected == 0 {
				return sql.ErrNoRows
			}

			comment := ""
			if payload.LComment != nil {
				comment = *payload.LComment
			}

			insert := tx.Exec(
				"INSERT INTO lineitem (l_orderkey, l_linenumber, l_extendedprice, l_discount, l_tax, l_shipdate, l_comment) VALUES (?, ?, ?, ?, ?, ?, ?)",
				oOrderkey, payload.LLinenumber, payload.LExtendedprice,
				payload.LDiscount, payload.LTax,
				payload.LShipdate, comment,
			)
			return insert.Error
		})

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"l_orderkey": oOrderkey, "l_linenumber": payload.LLinenumber})
	})

	r.POST("/api/orders", func(c *gin.Context) {
		var payload CreateOrderPayload
		if err := c.ShouldBindJSON(&payload); err != nil {
			payload = CreateOrderPayload{}
		}

		var oOrderkey int64
		var total float64

		err := db.Transaction(func	tx *gorm.DB) error {
			orderStatus := "O"
			if payload.OOrderstatus != nil {
				orderStatus = *payload.OOrderstatus
			}

			comment := ""
			if payload.OComment != nil {
				comment = *payload.OComment
			}

			insertOrder := tx.Exec(
				"INSERT INTO orders (o_orderstatus, o_totalprice, o_orderdate, o_comment) VALUES (?, 0, ?, ?)",
				orderStatus, payload.OOrderdate, comment,
			)
			if insertOrder.Error != nil {
				return insertOrder.Error
			}

			var rowID int64
			if err := tx.Raw("SELECT last_insert_rowid()").Scan(&rowID).Error; err != nil {
				return err
			}
			oOrderkey = rowID

			for _, item := range payload.Lineitems {
				itemComment := ""
				if item.LComment != nil {
					itemComment = *item.LComment
				}

				insertItem := tx.Exec(
					"INSERT INTO lineitem (l_orderkey, l_linenumber, l_extendedprice, l_discount, l_tax, l_shipdate, l_comment) VALUES (?, ?, ?, ?, ?, ?, ?)",
					oOrderkey, item.LLinenumber, item.LExtendedprice,
					item.LDiscount, item.LTax,
					item.LShipdate, itemComment,
				)
				if insertItem.Error != nil {
					return insertItem.Error
				}
			}

			var totalRow struct {
				Total float64 `gorm:"column:total"`
			}
			sumQuery := tx.Raw(
				"SELECT COALESCE(SUM(l_extendedprice * (1 - l_discount) * (1 + l_tax)), 0) AS total FROM lineitem WHERE l_orderkey = ?",
				oOrderkey,
			).Scan(&totalRow)
			if sumQuery.Error != nil {
				return sumQuery.Error
			}
			total = totalRow.Total

			updateOrder := tx.Exec(
				"UPDATE orders SET o_totalprice = ? WHERE o_orderkey = ?",
				total, oOrderkey,
			)
			return updateOrder.Error
		})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"o_orderkey": oOrderkey, "o_totalprice": total})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	if err := r.Run("0.0.0.0:" + port); err != nil {
		panic(err)
	}
}