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
	defer db.Close()

	router := gin.Default()

	router.DELETE("/api/customers/:c_custkey", deleteCustomer)
	router.GET("/api/orders/summary", ordersSummary)

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
		c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
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

	tx, err := db.Begin()
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM orders WHERE o_custkey = ?", cCustkey)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec("DELETE FROM customer WHERE c_custkey = ?", cCustkey)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusNoContent)
}

func ordersSummary(c *gin.Context) {
	/*
	   Return the summary of all existed historical orders
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
		"total_orders":  totalOrders,
		"total_revenue": totalRevenue,
	})
}