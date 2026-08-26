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
	sqlitegorm "gorm.io/gorm/logger"
)

type Customer struct {
	CCustkey int    `gorm:"column:c_custkey;primaryKey" json:"c_custkey"`
	CName    string `gorm:"column:c_name" json:"c_name"`
	CEmail   string `gorm:"column:c_email" json:"c_email"`
}

func (Customer) TableName() string {
	return "customer"
}

type Order struct {
	OOrderkey   int     `gorm:"column:o_orderkey;primaryKey" json:"o_orderkey"`
	OCustkey    int     `gorm:"column:o_custkey" json:"o_custkey"`
	OTotalprice float64 `gorm:"column:o_totalprice" json:"o_totalprice"`
	OOrderdate  string  `gorm:"column:o_orderdate" json:"o_orderdate"`
	OComment    string  `gorm:"column:o_comment" json:"o_comment"`
}

func (Order) TableName() string {
	return "orders"
}

func openDB() (*gorm.DB, error) {
	database := os.Getenv("DB_PATH")
	if database == "" {
		database = "app.db"
	}

	db, err := gorm.Open(sqlite.Open(database), &gorm.Config{
		Logger: sqlitegorm.Default.LogMode(sqlitegorm.Silent),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxOpenConns(1)

	if _, err := sqlDB.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		return nil, err
	}

	return db, nil
}

func main() {
	db, err := openDB()
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

		err = db.Raw(
			"SELECT c_custkey FROM customer WHERE c_custkey = ?",
			cCustkey,
		).Scan(&existing).Error

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
				return
			}
			c.Status(http.StatusInternalServerError)
			return
		}

		if existing.CCustkey == 0 {
			var count int64
			countErr := db.Raw(
				"SELECT COUNT(*) FROM customer WHERE c_custkey = ?",
				cCustkey,
			).Scan(&count).Error
			if countErr != nil {
				c.Status(http.StatusInternalServerError)
				return
			}
			if count == 0 {
				c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
				return
			}
		}

		res := db.Exec("DELETE FROM customer WHERE c_custkey = ?", cCustkey)
		if res.Error != nil {
			var sqliteErr interface{ Error() string }
			if errors.As(res.Error, &sqliteErr) {
				c.JSON(http.StatusConflict, gin.H{"error": "Cannot delete customer with existing orders"})
				return
			}
			c.Status(http.StatusInternalServerError)
			return
		}

		c.Status(http.StatusNoContent)
	})

	r.GET("/api/dashboard", func(c *gin.Context) {
		/*
		   Return dashboard based on all existed orders
		*/
		type DashboardRow struct {
			TotalOrders  int64   `json:"total_orders"`
			TotalRevenue float64 `json:"total_revenue"`
		}

		var row DashboardRow
		err := db.Raw(
			"SELECT COUNT(*) AS total_orders, " +
				"       COALESCE(SUM(o_totalprice), 0) AS total_revenue " +
				"FROM orders",
		).Scan(&row).Error
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
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