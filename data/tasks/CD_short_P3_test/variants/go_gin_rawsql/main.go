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

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	var err error
	databasePath := getEnv("DB_PATH", "app.db")
	db, err = sql.Open("sqlite", databasePath)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		panic(err)
	}

	app := gin.Default()

	app.DELETE("/api/customers/:c_custkey", deleteCustomer)
	app.GET("/api/orders/monthly-trend", monthlyTrend)

	port := getEnv("PORT", "5000")
	if _, err := strconv.Atoi(port); err != nil {
		port = "5000"
	}
	if err := app.Run("0.0.0.0:" + port); err != nil {
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
	if err = tx.Commit(); err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusNoContent)
}

func monthlyTrend(c *gin.Context) {
	/*
	   Return the trend of all existed orders per-month.
	*/
	rows, err := db.Query(
		"SELECT strftime('%Y-%m', o_orderdate) AS month, " +
			"       COUNT(*) AS order_count, " +
			"       SUM(o_totalprice) AS revenue " +
			"FROM orders " +
			"GROUP BY month " +
			"ORDER BY month",
	)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	result := make([]gin.H, 0)
	for rows.Next() {
		var month sql.NullString
		var orderCount int64
		var revenue sql.NullFloat64

		if err := rows.Scan(&month, &orderCount, &revenue); err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		item := gin.H{
			"month":       nil,
			"order_count": orderCount,
			"revenue":     nil,
		}
		if month.Valid {
			item["month"] = month.String
		}
		if revenue.Valid {
			item["revenue"] = revenue.Float64
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, result)
}