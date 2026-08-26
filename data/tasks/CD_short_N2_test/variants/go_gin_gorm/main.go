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

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "app.db"
	}

	gormDB, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		panic(err)
	}
	defer sqlDB.Close()

	app := gin.Default()

	app.DELETE("/api/customers/:c_custkey", func(c *gin.Context) {
		cCustkey, err := strconv.Atoi(c.Param("c_custkey"))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
			return
		}

		var existing sql.NullInt64
		row := gormDB.Raw(
			"SELECT c_custkey FROM customer WHERE c_custkey = ?",
			cCustkey,
		).Row()
		err = row.Scan(&existing)
		if err == sql.ErrNoRows || !existing.Valid {
			c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
			return
		}
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		tx := gormDB.Begin()
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
		/*
		   Return orders tied to active customers
		*/
		var orders []Order
		err := gormDB.Raw(
			"SELECT o_orderkey, o_custkey, o_totalprice, o_orderdate, o_comment " +
				"FROM orders WHERE o_custkey IN (SELECT c_custkey FROM customer) " +
				"ORDER BY o_orderdate",
		).Scan(&orders).Error
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.JSON(http.StatusOK, orders)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	if err := app.Run("0.0.0.0:" + port); err != nil {
		panic(err)
	}
}