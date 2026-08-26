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

type Customer struct {
	CCustkey int            `gorm:"column:c_custkey;primaryKey" json:"c_custkey"`
	CName    sql.NullString `gorm:"column:c_name" json:"c_name"`
	CEmail   sql.NullString `gorm:"column:c_email" json:"c_email"`
}

func (Customer) TableName() string {
	return "customer"
}

type Order struct {
	OOrderkey   int             `gorm:"column:o_orderkey;primaryKey" json:"o_orderkey"`
	OCustkey    sql.NullInt64   `gorm:"column:o_custkey" json:"o_custkey"`
	OTotalprice sql.NullFloat64 `gorm:"column:o_totalprice" json:"o_totalprice"`
	OOrderdate  sql.NullString  `gorm:"column:o_orderdate" json:"o_orderdate"`
	OComment    sql.NullString  `gorm:"column:o_comment" json:"o_comment"`
}

func (Order) TableName() string {
	return "orders"
}

type OrderResponse struct {
	OOrderkey   int      `json:"o_orderkey"`
	OCustkey    *int64   `json:"o_custkey"`
	OTotalprice *float64 `json:"o_totalprice"`
	OOrderdate  *string  `json:"o_orderdate"`
	OComment    *string  `json:"o_comment"`
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

	app := gin.Default()

	app.DELETE("/api/customers/:c_custkey", func(c *gin.Context) {
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

	app.GET("/api/orders", func(c *gin.Context) {
		/**
		Return orders tied to active customers
		*/
		var rows []Order
		err := db.Table("orders o").
			Select("o.o_orderkey, o.o_custkey, o.o_totalprice, o.o_orderdate, o.o_comment").
			Joins("JOIN customer c ON o.o_custkey = c.c_custkey").
			Order("o.o_orderdate").
			Scan(&rows).Error
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		resp := make([]OrderResponse, 0, len(rows))
		for _, r := range rows {
			item := OrderResponse{
				OOrderkey: r.OOrderkey,
			}
			if r.OCustkey.Valid {
				v := r.OCustkey.Int64
				item.OCustkey = &v
			}
			if r.OTotalprice.Valid {
				v := r.OTotalprice.Float64
				item.OTotalprice = &v
			}
			if r.OOrderdate.Valid {
				v := r.OOrderdate.String
				item.OOrderdate = &v
			}
			if r.OComment.Valid {
				v := r.OComment.String
				item.OComment = &v
			}
			resp = append(resp, item)
		}

		c.JSON(http.StatusOK, resp)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	if err := app.Run("0.0.0.0:" + port); err != nil {
		panic(err)
	}
}