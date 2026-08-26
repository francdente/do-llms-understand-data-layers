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

	if err = db.Ping(); err != nil {
		panic(err)
	}

	r := gin.Default()

	r.DELETE("/api/customers/:c_custkey", deleteCustomer)
	r.GET("/api/orders", listOrders)

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	if err := r.Run("0.0.0.0:" + port); err != nil {
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

func listOrders(c *gin.Context) {
	/**
	 * Return orders tied to active customers
	 */
	rows, err := db.Query(
		"SELECT o.o_orderkey, o.o_custkey, o.o_totalprice, o.o_orderdate, o.o_comment " +
			"FROM orders o JOIN customer c ON o.o_custkey = c.c_custkey " +
			"ORDER BY o.o_orderdate",
	)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	result := make([]map[string]interface{}, 0)
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

		row := map[string]interface{}{
			"o_orderkey":   oOrderkey,
			"o_custkey":    oCustkey,
			"o_totalprice": oTotalprice,
			"o_orderdate":  oOrderdate,
		}
		if oComment.Valid {
			row["o_comment"] = oComment.String
		} else {
			row["o_comment"] = nil
		}

		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, result)
}