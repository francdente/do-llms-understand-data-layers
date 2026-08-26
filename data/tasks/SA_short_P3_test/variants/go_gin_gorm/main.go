package main

import (
	"database/sql"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
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

	sqlDB, err := db.DB()
	if err != nil {
		panic(err)
	}
	defer sqlDB.Close()

	r := gin.Default()

	r.POST("/api/orders/:o_orderkey/lineitems/:l_linenumber/ship", func(c *gin.Context) {
		oOrderkey, err := strconv.Atoi(c.Param("o_orderkey"))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Line item not found"})
			return
		}
		lLinenumber, err := strconv.Atoi(c.Param("l_linenumber"))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Line item not found"})
			return
		}

		var existing struct {
			LOrderkey int `gorm:"column:l_orderkey"`
		}

		tx := db.Table("lineitem").
			Select("l_orderkey").
			Where("l_orderkey = ? AND l_linenumber = ?", oOrderkey, lLinenumber).
			Take(&existing)

		if tx.Error != nil {
			if tx.Error == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Line item not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": tx.Error.Error()})
			return
		}

		today := time.Now().Format("2006-01-02")

		updateTx := db.Table("lineitem").
			Where("l_orderkey = ? AND l_linenumber = ?", oOrderkey, lLinenumber).
			Update("l_shipdate", today)

		if updateTx.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": updateTx.Error.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"l_orderkey":   oOrderkey,
			"l_linenumber": lLinenumber,
			"l_shipdate":   today,
		})
	})

	r.GET("/api/orders/:o_orderkey", func(c *gin.Context) {
		oOrderkey, err := strconv.Atoi(c.Param("o_orderkey"))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
			return
		}

		type OrderRow struct {
			OOrderkey       int            `gorm:"column:o_orderkey" json:"o_orderkey"`
			OTotalprice     sql.NullFloat64 `gorm:"column:o_totalprice"`
			OOrderdate      sql.NullString  `gorm:"column:o_orderdate"`
			OLatestShipdate sql.NullString  `gorm:"column:o_latest_shipdate"`
			OComment        sql.NullString  `gorm:"column:o_comment"`
		}

		var row OrderRow
		tx := db.Table("orders").
			Select("o_orderkey, o_totalprice, o_orderdate, o_latest_shipdate, o_comment").
			Where("o_orderkey = ?", oOrderkey).
			Take(&row)

		if tx.Error != nil {
			if tx.Error == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": tx.Error.Error()})
			return
		}

		resp := gin.H{
			"o_orderkey": oOrderkey,
		}
		if row.OTotalprice.Valid {
			resp["o_totalprice"] = row.OTotalprice.Float64
		} else {
			resp["o_totalprice"] = nil
		}
		if row.OOrderdate.Valid {
			resp["o_orderdate"] = row.OOrderdate.String
		} else {
			resp["o_orderdate"] = nil
		}
		if row.OLatestShipdate.Valid {
			resp["o_latest_shipdate"] = row.OLatestShipdate.String
		} else {
			resp["o_latest_shipdate"] = nil
		}
		if row.OComment.Valid {
			resp["o_comment"] = row.OComment.String
		} else {
			resp["o_comment"] = nil
		}

		c.JSON(http.StatusOK, resp)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	if err := r.Run("0.0.0.0:" + port); err != nil {
		panic(err)
	}
}