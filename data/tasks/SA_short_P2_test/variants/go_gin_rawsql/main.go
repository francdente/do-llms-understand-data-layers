package main

import (
	"database/sql"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	_ "modernc.org/sqlite"
)

var DATABASE = getEnv("DB_PATH", "app.db")

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite", DATABASE)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func get_order(c *gin.Context) {
	oOrderkey, err := strconv.Atoi(c.Param("o_orderkey"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	db, err := getDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer db.Close()

	var o_orderkey int
	var o_orderstatus string
	var o_totalprice float64
	var o_orderdate string
	var o_comment string

	err = db.QueryRow(
		"SELECT o_orderkey, o_orderstatus, o_totalprice, o_orderdate, o_comment "+
			"FROM orders WHERE o_orderkey = ?",
		oOrderkey,
	).Scan(&o_orderkey, &o_orderstatus, &o_totalprice, &o_orderdate, &o_comment)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"o_orderkey":    o_orderkey,
		"o_orderstatus": o_orderstatus,
		"o_totalprice":  o_totalprice,
		"o_orderdate":   o_orderdate,
		"o_comment":     o_comment,
	})
}

func add_lineitem(c *gin.Context) {
	oOrderkey, err := strconv.Atoi(c.Param("o_orderkey"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		payload = map[string]interface{}{}
	}

	db, err := getDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var orderKey int
	err = tx.QueryRow(
		"SELECT o_orderkey FROM orders WHERE o_orderkey = ?", oOrderkey,
	).Scan(&orderKey)
	if err == sql.ErrNoRows {
		tx.Exec("ROLLBACK")
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	_, err = tx.Exec(
		"INSERT INTO lineitem (l_orderkey, l_linenumber, l_extendedprice, l_discount, l_tax, l_shipdate, l_comment) "+
			"VALUES (?, ?, ?, ?, ?, ?, ?)",
		oOrderkey,
		toInt(payload["l_linenumber"]),
		toFloat(payload["l_extendedprice"]),
		toFloat(payload["l_discount"]),
		toFloat(payload["l_tax"]),
		getNullableString(payload, "l_shipdate"),
		getStringDefault(payload, "l_comment", ""),
	)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"l_orderkey":   oOrderkey,
		"l_linenumber": toInt(payload["l_linenumber"]),
	})
}

func create_order(c *gin.Context) {
	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		payload = map[string]interface{}{}
	}

	db, err := getDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	res, err := tx.Exec(
		"INSERT INTO orders (o_orderstatus, o_totalprice, o_orderdate, o_comment) "+
			"VALUES (?, 0, ?, ?)",
		getStringDefault(payload, "o_orderstatus", "O"),
		getRequiredString(payload["o_orderdate"]),
		getStringDefault(payload, "o_comment", ""),
	)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	lastID, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	o_orderkey := int(lastID)

	lineitemsRaw := payload["lineitems"].([]interface{})
	for _, itemRaw := range lineitemsRaw {
		item := itemRaw.(map[string]interface{})
		_, err = tx.Exec(
			"INSERT INTO lineitem (l_orderkey, l_linenumber, l_extendedprice, "+
				"l_discount, l_tax, l_shipdate, l_comment) "+
				"VALUES (?, ?, ?, ?, ?, ?, ?)",
			o_orderkey,
			toInt(item["l_linenumber"]),
			toFloat(item["l_extendedprice"]),
			toFloat(item["l_discount"]),
			toFloat(item["l_tax"]),
			getNullableString(item, "l_shipdate"),
			getStringDefault(item, "l_comment", ""),
		)
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	var total float64
	err = tx.QueryRow(
		"SELECT COALESCE(SUM(l_extendedprice * (1 - l_discount) * (1 + l_tax)), 0) AS total "+
			"FROM lineitem WHERE l_orderkey = ?",
		o_orderkey,
	).Scan(&total)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	_, err = tx.Exec(
		"UPDATE orders SET o_totalprice = ? WHERE o_orderkey = ?",
		total, o_orderkey,
	)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"o_orderkey":   o_orderkey,
		"o_totalprice": total,
	})
}

func toInt(v interface{}) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	default:
		return 0
	}
}

func toFloat(v interface{}) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case int64:
		return float64(t)
	default:
		return 0
	}
}

func getStringDefault(m map[string]interface{}, key, def string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return def
	}
	s, ok := v.(string)
	if !ok {
		return def
	}
	return s
}

func getNullableString(m map[string]interface{}, key string) interface{} {
	v, ok := m[key]
	if !ok || v == nil {
		return nil
	}
	s, ok := v.(string)
	if !ok {
		return nil
	}
	return s
}

func getRequiredString(v interface{}) string {
	s, _ := v.(string)
	return s
}

func main() {
	app := gin.Default()

	app.GET("/api/orders/:o_orderkey", get_order)
	app.POST("/api/orders/:o_orderkey/lineitems", add_lineitem)
	app.POST("/api/orders", create_order)

	port := getEnv("PORT", "5000")
	app.Run("0.0.0.0:" + port)
}