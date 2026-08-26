package main

import (
	"database/sql"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	_ "modernc.org/sqlite"
)

var db *sql.DB
var databasePath = getEnv("DB_PATH", "app.db")

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getDB() *sql.DB {
	return db
}

func closeDB() {
	if db != nil {
		_ = db.Close()
	}
}

func shipLineitem(c *gin.Context) {
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

	db := getDB()
	var existing int
	err = db.QueryRow(
		"SELECT l_orderkey FROM lineitem WHERE l_orderkey = ? AND l_linenumber = ?",
		oOrderkey, lLinenumber,
	).Scan(&existing)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Line item not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	today := time.Now().Format("2006-01-02")
	_, err = db.Exec(
		"UPDATE lineitem SET l_shipdate = ? WHERE l_orderkey = ? AND l_linenumber = ?",
		today, oOrderkey, lLinenumber,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"l_orderkey":   oOrderkey,
		"l_linenumber": lLinenumber,
		"l_shipdate":   today,
	})
}

func getOrder(c *gin.Context) {
	oOrderkey, err := strconv.Atoi(c.Param("o_orderkey"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	db := getDB()
	var o_orderkey int
	var o_totalprice sql.NullFloat64
	var o_orderdate sql.NullString
	var o_latest_shipdate sql.NullString
	var o_comment sql.NullString

	err = db.QueryRow(
		"SELECT o_orderkey, o_totalprice, o_orderdate, o_latest_shipdate, o_comment FROM orders WHERE o_orderkey = ?",
		oOrderkey,
	).Scan(&o_orderkey, &o_totalprice, &o_orderdate, &o_latest_shipdate, &o_comment)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := gin.H{
		"o_orderkey": o_orderkey,
	}
	if o_totalprice.Valid {
		resp["o_totalprice"] = o_totalprice.Float64
	} else {
		resp["o_totalprice"] = nil
	}
	if o_orderdate.Valid {
		resp["o_orderdate"] = o_orderdate.String
	} else {
		resp["o_orderdate"] = nil
	}
	if o_latest_shipdate.Valid {
		resp["o_latest_shipdate"] = o_latest_shipdate.String
	} else {
		resp["o_latest_shipdate"] = nil
	}
	if o_comment.Valid {
		resp["o_comment"] = o_comment.String
	} else {
		resp["o_comment"] = nil
	}

	c.JSON(http.StatusOK, resp)
}

func main() {
	var err error
	db, err = sql.Open("sqlite", databasePath)
	if err != nil {
		panic(err)
	}
	defer closeDB()

	router := gin.Default()

	router.POST("/api/orders/:o_orderkey/lineitems/:l_linenumber/ship", shipLineitem)
	router.GET("/api/orders/:o_orderkey", getOrder)

	port := getEnv("PORT", "5000")
	if err := router.Run("0.0.0.0:" + port); err != nil {
		panic(err)
	}
}