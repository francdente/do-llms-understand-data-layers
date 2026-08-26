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
	router.GET("/api/orders", listOrders)

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

	tx, err := db.Begin()
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var existing int
	err = tx.QueryRow(
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

func listOrders(c *gin.Context) {
	/*
	   Return orders tied to active customers
	*/
	rows, err := db.Query(
		"SELECT o_orderkey, o_custkey, o_totalprice, o_orderdate, o_comment " +
			"FROM orders WHERE o_custkey IN (SELECT c_custkey FROM customer) " +
			"ORDER BY o_orderdate",
	)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	result := make([]gin.H, 0)
	for rows.Next() {
		var oOrderkey int
		var oCustkey int
		var oTotalprice float64
		var oOrderdate string
		var oComment sql.NullString

		if err := rows.Scan(&oOrderkey, &oCustkey, &oTotalprice, &oOrderdate, &oComment); err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		var comment interface{}
		if oComment.Valid {
			comment = oComment.String
		} else {
			comment = nil
		}

		result = append(result, gin.H{
			"o_orderkey":   oOrderkey,
			"o_custkey":    oCustkey,
			"o_totalprice": oTotalprice,
			"o_orderdate":  oOrderdate,
			"o_comment":    comment,
		})
	}

	if err := rows.Err(); err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, result)
}