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

func main() {
	r := gin.Default()

	database := os.Getenv("DB_PATH")
	if database == "" {
		database = "app.db"
	}

	db, err := gorm.Open(sqlite.Open(database), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	// @app.route("/api/customers/<int:c_custkey>", methods=["DELETE"])
	r.DELETE("/api/customers/:c_custkey", func(c *gin.Context) {
		deleteCustomer(c, db)
	})

	// @app.route("/api/orders/summary", methods=["GET"])
	r.GET("/api/orders/summary", func(c *gin.Context) {
		ordersSummary(c, db)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	if err := r.Run("0.0.0.0:" + port); err != nil {
		panic(err)
	}
}

func deleteCustomer(c *gin.Context, db *gorm.DB) {
	cCustkey, err := strconv.Atoi(c.Param("c_custkey"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
		return
	}

	var existing struct {
		CCustkey int `gorm:"column:c_custkey"`
	}

	result := db.Raw(
		"SELECT c_custkey FROM customer WHERE c_custkey = ?",
		cCustkey,
	).Scan(&existing)

	if result.Error != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
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
}

func ordersSummary(c *gin.Context, db *gorm.DB) {
	/**
	    Return the summary of all existed historical orders
	*/
	type Summary struct {
		TotalOrders  int             `json:"total_orders" gorm:"column:total_orders"`
		TotalRevenue sql.NullFloat64 `gorm:"column:total_revenue"`
	}

	var row Summary
	result := db.Raw(
		"SELECT COUNT(*) AS total_orders, " +
			"       COALESCE(SUM(o_totalprice), 0) AS total_revenue " +
			"FROM orders",
	).Scan(&row)

	if result.Error != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	totalRevenue := 0.0
	if row.TotalRevenue.Valid {
		totalRevenue = row.TotalRevenue.Float64
	}

	c.JSON(http.StatusOK, gin.H{
		"total_orders":  row.TotalOrders,
		"total_revenue": totalRevenue,
	})
}