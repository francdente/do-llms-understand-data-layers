package main

import (
	"database/sql"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
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

type OrderResult struct {
	OOrderkey    int     `json:"o_orderkey"`
	OOrderstatus string  `json:"o_orderstatus"`
	OOrderdate   string  `json:"o_orderdate"`
	OComment     string  `json:"o_comment"`
	OTotalprice  float64 `json:"o_totalprice"`
}

type LineitemUpdatePayload struct {
	LExtendedprice float64 `json:"l_extendedprice"`
	LDiscount      float64 `json:"l_discount"`
	LTax           float64 `json:"l_tax"`
}

type LineitemResult struct {
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

	sqlDB, err := db.DB()
	if err != nil {
		panic(err)
	}
	defer sqlDB.Close()

	r := gin.Default()

	r.GET("/api/orders/:o_orderkey", func(c *gin.Context) {
		oOrderkey, err := strconv.Atoi(c.Param("o_orderkey"))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
			return
		}

		tx := db.Begin()
		if tx.Error != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		var row Order
		err = tx.Raw(
			"SELECT o_orderkey, o_orderstatus, o_orderdate, o_comment FROM orders WHERE o_orderkey = ?",
			oOrderkey,
		).Scan(&row).Error
		if err != nil {
			tx.Rollback()
			c.Status(http.StatusInternalServerError)
			return
		}

		if tx.RowsAffected == 0 {
			tx.Exec("ROLLBACK")
			c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
			return
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
			c.Status(http.StatusInternalServerError)
			return
		}

		if err := tx.Exec("COMMIT").Error; err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		result := OrderResult{
			OOrderkey:    row.OOrderkey,
			OOrderstatus: row.OOrderstatus,
			OOrderdate:   row.OOrderdate,
			OComment:     row.OComment,
			OTotalprice:  totalRow.Total,
		}
		c.JSON(http.StatusOK, result)
	})

	r.PUT("/api/orders/:o_orderkey/lineitems/:l_linenumber", func(c *gin.Context) {
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

		var payload LineitemUpdatePayload
		if err := c.ShouldBindJSON(&payload); err != nil {
			payload = LineitemUpdatePayload{}
		}

		tx := db.Begin()
		if tx.Error != nil {
			c.Status(http.StatusInternalServerError)
			return
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
			c.Status(http.StatusInternalServerError)
			return
		}

		if tx.RowsAffected == 0 {
			tx.Exec("ROLLBACK")
			c.JSON(http.StatusNotFound, gin.H{"error": "Line item not found"})
			return
		}

		updateResult := tx.Exec(
			"UPDATE lineitem SET l_extendedprice = ?, l_discount = ?, l_tax = ? WHERE l_orderkey = ? AND l_linenumber = ?",
			payload.LExtendedprice, payload.LDiscount, payload.LTax, oOrderkey, lLinenumber,
		)
		if updateResult.Error != nil {
			tx.Rollback()
			c.Status(http.StatusInternalServerError)
			return
		}

		var row LineitemResult
		err = tx.Raw(
			"SELECT l_orderkey, l_linenumber, l_extendedprice, l_discount, l_tax FROM lineitem WHERE l_orderkey = ? AND l_linenumber = ?",
			oOrderkey, lLinenumber,
		).Scan(&row).Error
		if err != nil {
			tx.Rollback()
			c.Status(http.StatusInternalServerError)
			return
		}

		if err := tx.Commit().Error; err != nil && err != sql.ErrTxDone {
			c.Status(http.StatusInternalServerError)
			return
		}

		c.JSON(http.StatusOK, row)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	if err := r.Run("0.0.0.0:" + port); err != nil {
		panic(err)
	}
}