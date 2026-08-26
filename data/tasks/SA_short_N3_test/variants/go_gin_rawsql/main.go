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

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	var err error
	database := getEnv("DB_PATH", "app.db")
	db, err = sql.Open("sqlite", database)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	router := gin.Default()

	router.POST("/api/orders/:o_orderkey/lineitems/:l_linenumber/ship", shipLineitem)
	router.GET("/api/orders/:o_orderkey", getOrder)

	port := getEnv("PORT", "5000")
	if err := router.Run("0.0.0.0:" + port); err != nil {
		panic(err)
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

	tx, err := db.Begin()
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var existing int
	err = tx.QueryRow(
		"SELECT l_orderkey FROM lineitem WHERE l_orderkey = ? AND l_linenumber = ?",
		oOrderkey, lLinenumber,
	).Scan(&existing)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Line item not found"})
		return
	}
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	today := time.Now().Format("2006-01-02")

	_, err = tx.Exec(
		"UPDATE lineitem SET l_shipdate = ? WHERE l_orderkey = ? AND l_linenumber = ?",
		today, oOrderkey, lLinenumber,
	)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	var latest sql.NullString
	err = tx.QueryRow(
		"SELECT MAX(l_shipdate) AS latest FROM lineitem WHERE l_orderkey = ?",
		oOrderkey,
	).Scan(&latest)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	var latestValue interface{}
	if latest.Valid {
		latestValue = latest.String
	} else {
		latestValue = nil
	}

	_, err = tx.Exec(
		"UPDATE orders SET o_latest_shipdate = ? WHERE o_orderkey = ?",
		latestValue, oOrderkey,
	)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		c.Status(http.StatusInternalServerError)
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

	var oOrderkeyVal int
	var oTotalprice sql.NullFloat64
	var oOrderdate sql.NullString
	var oLatestShipdate sql.NullString
	var oComment sql.NullString

	err = db.QueryRow(
		"SELECT o_orderkey, o_totalprice, o_orderdate, o_latest_shipdate, o_comment FROM orders WHERE o_orderkey = ?",
		oOrderkey,
	).Scan(&oOrderkeyVal, &oTotalprice, &oOrderdate, &oLatestShipdate, &oComment)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	resp := gin.H{
		"o_orderkey":       oOrderkeyVal,
		"o_totalprice":     nil,
		"o_orderdate":      nil,
		"o_latest_shipdate": nil,
		"o_comment":        nil,
	}

	if oTotalprice.Valid {
		resp["o_totalprice"] = oTotalprice.Float64
	}
	if oOrderdate.Valid {
		resp["o_orderdate"] = oOrderdate.String
	}
	if oLatestShipdate.Valid {
		resp["o_latest_shipdate"] = oLatestShipdate.String
	}
	if oComment.Valid {
		resp["o_comment"] = oComment.String
	}

	c.JSON(http.StatusOK, resp)
}