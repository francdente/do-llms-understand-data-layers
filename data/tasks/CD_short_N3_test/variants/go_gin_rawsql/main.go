package main

import (
	"database/sql"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	_ "modernc.org/sqlite"
)

var db *sql.DB

func main() {
	database := os.Getenv("DB_PATH")
	if database == "" {
		database = "app.db"
	}

	var err error
	db, err = sql.Open("sqlite", database)
	if err != nil {
		panic(err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		panic(err)
	}

	router := gin.Default()

	router.DELETE("/api/customers/:c_custkey", deleteCustomer)
	router.GET("/api/dashboard", dashboard)

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	if err := router.Run("0.0.0.0:" + port); err != nil {
		panic(err)
	}
}

func deleteCustomer(c *gin.Context) {
	cCustkey, err := strconv.Atoi(c.Param("c_custkey"))
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	var existing int
	err = db.QueryRow(
		"SELECT c_custkey FROM customer WHERE c_custkey = ?",
		cCustkey,
	).Scan(&existing)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
		return
	}
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	_, err = db.Exec("DELETE FROM customer WHERE c_custkey = ?", cCustkey)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Cannot delete customer with existing orders"})
		return
	}

	c.Status(http.StatusNoContent)
}

func dashboard(c *gin.Context) {
	/**
	 * Return dashboard based on all existed orders
	 */
	var totalOrders int64
	var totalRevenue float64

	err := db.QueryRow(
		"SELECT COUNT(*) AS total_orders, " +
			"       COALESCE(SUM(o_totalprice), 0) AS total_revenue " +
			"FROM orders",
	).Scan(&totalOrders, &totalRevenue)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total_orders":   totalOrders,
		"total_revenue":  totalRevenue,
	})
}