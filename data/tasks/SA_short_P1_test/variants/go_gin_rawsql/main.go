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

func openDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite", DATABASE)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func getOrder(c *gin.Context) {
	oOrderkey, err := strconv.Atoi(c.Param("o_orderkey"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	db, err := openDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer db.Close()

	row := db.QueryRow(
		"SELECT o_orderkey, o_orderstatus, o_totalprice, o_orderdate, o_comment "+
			"FROM orders WHERE o_orderkey = ?",
		oOrderkey,
	)

	var orderkey int
	var orderstatus string
	var totalprice float64
	var orderdate string
	var comment string

	err = row.Scan(&orderkey, &orderstatus, &totalprice, &orderdate, &comment)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"o_orderkey":    orderkey,
		"o_orderstatus": orderstatus,
		"o_totalprice":  totalprice,
		"o_orderdate":   orderdate,
		"o_comment":     comment,
	})
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

	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		payload = map[string]interface{}{}
	}

	db, err := openDB()
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

	existing := tx.QueryRow(
		"SELECT l_orderkey FROM lineitem WHERE l_orderkey = ? AND l_linenumber = ?",
		oOrderkey, lLinenumber,
	)
	var existingOrderkey int
	err = existing.Scan(&existingOrderkey)
	if err == sql.ErrNoRows {
		_ = tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "Line item not found"})
		return
	}
	if err != nil {
		_ = tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	_, err = tx.Exec(
		"UPDATE lineitem SET l_extendedprice = ?, l_discount = ?, l_tax = ? "+
			"WHERE l_orderkey = ? AND l_linenumber = ?",
		payload["l_extendedprice"], payload["l_discount"], payload["l_tax"],
		oOrderkey, lLinenumber,
	)
	if err != nil {
		_ = tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	row := tx.QueryRow(
		"SELECT l_orderkey, l_linenumber, l_extendedprice, l_discount, l_tax "+
			"FROM lineitem WHERE l_orderkey = ? AND l_linenumber = ?",
		oOrderkey, lLinenumber,
	)

	var lOrderkey int
	var lnum int
	var extendedprice float64
	var discount float64
	var tax float64

	err = row.Scan(&lOrderkey, &lnum, &extendedprice, &discount, &tax)
	if err != nil {
		_ = tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"l_orderkey":      lOrderkey,
		"l_linenumber":    lnum,
		"l_extendedprice": extendedprice,
		"l_discount":      discount,
		"l_tax":           tax,
	})
}

func createOrder(c *gin.Context) {
	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		payload = map[string]interface{}{}
	}

	db, err := openDB()
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

	oOrderstatus := "O"
	if v, ok := payload["o_orderstatus"]; ok && v != nil {
		if s, ok := v.(string); ok {
			oOrderstatus = s
		}
	}

	oComment := ""
	if v, ok := payload["o_comment"]; ok && v != nil {
		if s, ok := v.(string); ok {
			oComment = s
		}
	}

	oOrderdate, _ := payload["o_orderdate"].(string)

	cur, err := tx.Exec(
		"INSERT INTO orders (o_orderstatus, o_totalprice, o_orderdate, o_comment) "+
			"VALUES (?, 0, ?, ?)",
		oOrderstatus, oOrderdate, oComment,
	)
	if err != nil {
		_ = tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	lastID, err := cur.LastInsertId()
	if err != nil {
		_ = tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	oOrderkey := lastID

	lineitemsRaw, _ := payload["lineitems"].([]interface{})
	for _, itemRaw := range lineitemsRaw {
		item, _ := itemRaw.(map[string]interface{})

		lLinenumber := item["l_linenumber"]
		lExtendedprice := item["l_extendedprice"]
		lDiscount := item["l_discount"]
		lTax := item["l_tax"]

		var lShipdate interface{}
		if v, ok := item["l_shipdate"]; ok {
			lShipdate = v
		} else {
			lShipdate = nil
		}

		lComment := ""
		if v, ok := item["l_comment"]; ok && v != nil {
			if s, ok := v.(string); ok {
				lComment = s
			}
		}

		_, err = tx.Exec(
			"INSERT INTO lineitem (l_orderkey, l_linenumber, l_extendedprice, "+
				"l_discount, l_tax, l_shipdate, l_comment) "+
				"VALUES (?, ?, ?, ?, ?, ?, ?)",
			oOrderkey, lLinenumber, lExtendedprice,
			lDiscount, lTax, lShipdate, lComment,
		)
		if err != nil {
			_ = tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	row := tx.QueryRow(
		"SELECT COALESCE(SUM(l_extendedprice * (1 - l_discount) * (1 + l_tax)), 0) AS total "+
			"FROM lineitem WHERE l_orderkey = ?",
		oOrderkey,
	)

	var total float64
	if err := row.Scan(&total); err != nil {
		_ = tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	_, err = tx.Exec(
		"UPDATE orders SET o_totalprice = ? WHERE o_orderkey = ?",
		total, oOrderkey,
	)
	if err != nil {
		_ = tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"o_orderkey":   oOrderkey,
		"o_totalprice": total,
	})
}

func main() {
	app := gin.Default()

	app.GET("/api/orders/:o_orderkey", getOrder)
	app.PUT("/api/orders/:o_orderkey/lineitems/:l_linenumber", updateLineitem)
	app.POST("/api/orders", createOrder)

	port := getEnv("PORT", "5000")
	_ = app.Run("0.0.0.0:" + port)
}