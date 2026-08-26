package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	_ "modernc.org/sqlite"
)

var db *sql.DB

func getDB() *sql.DB {
	return db
}

func getOrder(c *gin.Context) {
	oOrderkey, err := strconv.Atoi(c.Param("o_orderkey"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	db := getDB()
	tx, err := db.Begin()
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	var o_orderkey int
	var o_orderstatus string
	var o_orderdate string
	var o_comment string

	row := tx.QueryRow(
		"SELECT o_orderkey, o_orderstatus, o_orderdate, o_comment FROM orders WHERE o_orderkey = ?",
		oOrderkey,
	)
	err = row.Scan(&o_orderkey, &o_orderstatus, &o_orderdate, &o_comment)
	if err == sql.ErrNoRows {
		_ = tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}
	if err != nil {
		_ = tx.Rollback()
		c.Status(http.StatusInternalServerError)
		return
	}

	var total float64
	err = tx.QueryRow(
		"SELECT COALESCE(SUM(l_extendedprice * (1 - l_discount) * (1 + l_tax)), 0) AS total FROM lineitem WHERE l_orderkey = ?",
		oOrderkey,
	).Scan(&total)
	if err != nil {
		_ = tx.Rollback()
		c.Status(http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	result := gin.H{
		"o_orderkey":    o_orderkey,
		"o_orderstatus": o_orderstatus,
		"o_orderdate":   o_orderdate,
		"o_comment":     o_comment,
		"o_totalprice":  total,
	}
	c.JSON(http.StatusOK, result)
}

func updateLineitem(c *gin.Context) {
	oOrderkey, err := strconv.Atoi(c.Param("o_orderkey"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Line item not found"})
		return
	}
	lLinenumber, err := strconv.Atoi(c.Param("l_linenumber"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Line item not found"})
		return
	}

	payload := map[string]interface{}{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		payload = map[string]interface{}{}
	}

	db := getDB()
	tx, err := db.Begin()
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	var existing int
	err = tx.QueryRow(
		"SELECT l_orderkey FROM lineitem WHERE l_orderkey = ? AND l_linenumber = ?",
		oOrderkey, lLinenumber,
	).Scan(&existing)
	if err == sql.ErrNoRows {
		_ = tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "Line item not found"})
		return
	}
	if err != nil {
		_ = tx.Rollback()
		c.Status(http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec(
		"UPDATE lineitem SET l_extendedprice = ?, l_discount = ?, l_tax = ? WHERE l_orderkey = ? AND l_linenumber = ?",
		payload["l_extendedprice"], payload["l_discount"], payload["l_tax"],
		oOrderkey, lLinenumber,
	)
	if err != nil {
		_ = tx.Rollback()
		c.Status(http.StatusInternalServerError)
		return
	}

	var l_orderkey int
	var l_linenumber int
	var l_extendedprice float64
	var l_discount float64
	var l_tax float64

	err = tx.QueryRow(
		"SELECT l_orderkey, l_linenumber, l_extendedprice, l_discount, l_tax FROM lineitem WHERE l_orderkey = ? AND l_linenumber = ?",
		oOrderkey, lLinenumber,
	).Scan(&l_orderkey, &l_linenumber, &l_extendedprice, &l_discount, &l_tax)
	if err != nil {
		_ = tx.Rollback()
		c.Status(http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"l_orderkey":      l_orderkey,
		"l_linenumber":    l_linenumber,
		"l_extendedprice": l_extendedprice,
		"l_discount":      l_discount,
		"l_tax":           l_tax,
	})
}

func main() {
	database := os.Getenv("DB_PATH")
	if database == "" {
		database = "app.db"
	}

	var err error
	db, err = sql.Open("sqlite", database)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	router := gin.Default()
	router.GET("/api/orders/:o_orderkey", getOrder)
	router.PUT("/api/orders/:o_orderkey/lineitems/:l_linenumber", updateLineitem)

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	if err := router.Run("0.0.0.0:" + port); err != nil {
		log.Fatal(err)
	}
}