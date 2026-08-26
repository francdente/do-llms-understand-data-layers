package main

import (
	"database/sql"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Customer struct {
	CCustkey int    `gorm:"column:c_custkey;primaryKey"`
	CName    string `gorm:"column:c_name"`
	CEmail   string `gorm:"column:c_email"`
}

func (Customer) TableName() string {
	return "customer"
}

type Order struct {
	OOrderkey   int     `gorm:"column:o_orderkey;primaryKey"`
	OCustkey    int     `gorm:"column:o_custkey"`
	OTotalprice float64 `gorm:"column:o_totalprice"`
	OOrderdate  string  `gorm:"column:o_orderdate"`
	OComment    string  `gorm:"column:o_comment"`
}

func (Order) TableName() string {
	return "orders"
}

type MonthlyTrendRow struct {
	Month      sql.NullString  `gorm:"column:month" json:"month"`
	OrderCount int64           `gorm:"column:order_count" json:"order_count"`
	Revenue    sql.NullFloat64 `gorm:"column:revenue" json:"revenue"`
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
		var customer Customer
		result := db.Select("c_custkey").Where("c_custkey = ?", c.Param("c_custkey")).First(&customer)
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

		if err := tx.Exec("DELETE FROM orders WHERE o_custkey = ?", c.Param("c_custkey")).Error; err != nil {
			tx.Rollback()
			c.Status(http.StatusInternalServerError)
			return
		}
		if err := tx.Exec("DELETE FROM customer WHERE c_custkey = ?", c.Param("c_custkey")).Error; err != nil {
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

	app.GET("/api/orders/monthly-trend", func(c *gin.Context) {
		/*
		   Return the trend of all existed orders per-month.
		*/
		var rows []MonthlyTrendRow
		err := db.Raw(
			"SELECT strftime('%Y-%m', o_orderdate) AS month, " +
				"       COUNT(*) AS order_count, " +
				"       SUM(o_totalprice) AS revenue " +
				"FROM orders " +
				"GROUP BY month " +
				"ORDER BY month",
		).Scan(&rows).Error
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		response := make([]gin.H, 0, len(rows))
		for _, r := range rows {
			var month interface{}
			if r.Month.Valid {
				month = r.Month.String
			} else {
				month = nil
			}

			var revenue interface{}
			if r.Revenue.Valid {
				revenue = r.Revenue.Float64
			} else {
				revenue = nil
			}

			response = append(response, gin.H{
				"month":       month,
				"order_count": r.OrderCount,
				"revenue":     revenue,
			})
		}

		c.JSON(http.StatusOK, response)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	if err := app.Run("0.0.0.0:" + port); err != nil {
		panic(err)
	}
}