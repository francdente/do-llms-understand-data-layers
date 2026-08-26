package main

import (
	"net/http"
	"os"
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

	r := gin.Default()

	r.POST("/api/orders/:o_orderkey/lineitems/:l_linenumber/ship", func(c *gin.Context) {
		var path struct {
			OOrderkey   int `uri:"o_orderkey" binding:"required"`
			LLinenumber int `uri:"l_linenumber" binding:"required"`
		}
		if err := c.ShouldBindUri(&path); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Bad request"})
			return
		}

		var existing struct {
			LOrderkey int `gorm:"column:l_orderkey"`
		}
		err := db.Table("lineitem").
			Select("l_orderkey").
			Where("l_orderkey = ? AND l_linenumber = ?", path.OOrderkey, path.LLinenumber).
			Take(&existing).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Line item not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		today := time.Now().Format("2006-01-02")

		tx := db.Begin()
		if tx.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		if err := tx.Table("lineitem").
			Where("l_orderkey = ? AND l_linenumber = ?", path.OOrderkey, path.LLinenumber).
			Update("l_shipdate", today).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		var latest struct {
			Latest *string `gorm:"column:latest"`
		}
		if err := tx.Table("lineitem").
			Select("MAX(l_shipdate) AS latest").
			Where("l_orderkey = ?", path.OOrderkey).
			Take(&latest).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		if err := tx.Table("orders").
			Where("o_orderkey = ?", path.OOrderkey).
			Update("o_latest_shipdate", latest.Latest).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		if err := tx.Commit().Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"l_orderkey":   path.OOrderkey,
			"l_linenumber": path.LLinenumber,
			"l_shipdate":   today,
		})
	})

	r.GET("/api/orders/:o_orderkey", func(c *gin.Context) {
		var path struct {
			OOrderkey int `uri:"o_orderkey" binding:"required"`
		}
		if err := c.ShouldBindUri(&path); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Bad request"})
			return
		}

		var row Order
		err := db.Table("orders").
			Select("o_orderkey, o_totalprice, o_orderdate, o_latest_shipdate, o_comment").
			Where("o_orderkey = ?", path.OOrderkey).
			Take(&row).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
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