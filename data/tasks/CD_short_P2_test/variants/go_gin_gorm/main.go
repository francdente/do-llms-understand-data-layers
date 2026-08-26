package main

import (
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Customer struct {
	CCustkey int     `gorm:"column:c_custkey;primaryKey" json:"c_custkey"`
	CName    *string `gorm:"column:c_name" json:"c_name"`
	CEmail   *string `gorm:"column:c_email" json:"c_email"`
}

func (Customer) TableName() string {
	return "customer"
}

type Order struct {
	OOrderkey   int      `gorm:"column:o_orderkey;primaryKey" json:"o_orderkey"`
	OCustkey    *int     `gorm:"column:o_custkey" json:"o_custkey"`
	OTotalprice *float64 `gorm:"column:o_totalprice" json:"o_totalprice"`
	OOrderdate  *string  `gorm:"column:o_orderdate" json:"o_orderdate"`
	OComment    *string  `gorm:"column:o_comment" json:"o_comment"`
}

func (Order) TableName() string {
	return "orders"
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

	r.DELETE("/api/customers/:c_custkey", func(c *gin.Context) {
		cCustkey, err := strconv.Atoi(c.Param("c_custkey"))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
			return
		}

		var existing struct {
			CCustkey int `gorm:"column:c_custkey"`
		}
		result := db.Table("customer").Select("c_custkey").Where("c_custkey = ?", cCustkey).Take(&existing)
		if result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
				return
			}
			c.Status(http.StatusInternalServerError)
			return
		}

		tx := db.Begin()
		if tx.Error != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		if err := tx.Exec("DELETE FROM orders WHERE o_custkey = ?", cCustkey).Error; err != nil {
			tx.Rollback()
			c.Status(http.StatusInternalServerError)
			return
		}
		if err := tx.Exec("DELETE FROM customer WHERE c_custkey = ?", cCustkey).Error; err != nil {
			tx.Rollback()
			c.Status(http.StatusInternalServerError)
			return
		}
		if err := tx.Commit().Error; err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		c.Status(http.StatusNoContent)
	})

	r.GET("/api/orders/history", func(c *gin.Context) {
		/**
		Return the history of all existed orders
		*/
		var rows []Order
		if err := db.Table("orders").
			Select("o_orderkey, o_custkey, o_totalprice, o_orderdate, o_comment").
			Order("o_orderdate").
			Find(&rows).Error; err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.JSON(http.StatusOK, rows)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	if err := r.Run("0.0.0.0:" + port); err != nil {
		panic(err)
	}
}