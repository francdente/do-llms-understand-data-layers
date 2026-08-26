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
	row := db.QueryRow(
		"SELECT o_orderkey, o_orderstatus, o_totalprice, o_orderdate, o_comment "+
			"FROM orders WHERE o_orderkey = ?",
		oOrderkey,
	)

	var orderkey int
	var orderstatus string
	var totalprice float64
	var orderdate string
	var comment sql.NullString

	err = row.Scan(&orderkey, &orderstatus, &totalprice, &orderdate, &comment)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var commentVal interface{}
	if comment.Valid {
		commentVal = comment.String
	} else {
		commentVal = nil
	}

	c.JSON(http.StatusOK, gin.H{
		"o_orderkey":    orderkey,
		"o_orderstatus": orderstatus,
		"o_totalprice":  totalprice,
		"o_orderdate":   orderdate,
		"o_comment":     commentVal,
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

	db := getDB()
	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	row := tx.QueryRow(
		"SELECT l_orderkey FROM lineitem WHERE l_orderkey = ? AND l_linenumber = ?",
		oOrderkey, lLinenumber,
	)
	var existing int
	err = row.Scan(&existing)
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

	lExtendedprice := payload["l_extendedprice"]
	lDiscount := payload["l_discount"]
	lTax := payload["l_tax"]

	_, err = tx.Exec(
		"UPDATE lineitem SET l_extendedprice = ?, l_discount = ?, l_tax = ? "+
			"WHERE l_orderkey = ? AND l_linenumber = ?",
		lExtendedprice, lDiscount, lTax, oOrderkey, lLinenumber,
	)
	if err != nil {
		_ = tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	_, err = tx.Exec(
		"UPDATE orders SET o_totalprice = "+
			"(SELECT COALESCE(SUM(l_extendedprice * (1 - l_discount) * (1 + l_tax)), 0) "+
			" FROM lineitem WHERE l_orderkey = ?) "+
			"WHERE o_orderkey = ?",
		oOrderkey, oOrderkey,
	)
	if err != nil {
		_ = tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	row = tx.QueryRow(
		"SELECT l_orderkey, l_linenumber, l_extendedprice, l_discount, l_tax "+
			"FROM lineitem WHERE l_orderkey = ? AND l_linenumber = ?",
		oOrderkey, lLinenumber,
	)

	var outOrderkey int
	var outLinenumber int
	var outExtendedprice float64
	var outDiscount float64
	var outTax float64

	err = row.Scan(&outOrderkey, &outLinenumber, &outExtendedprice, &outDiscount, &outTax)
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
		"l_orderkey":      outOrderkey,
		"l_linenumber":    outLinenumber,
		"l_extendedprice": outExtendedprice,
		"l_discount":      outDiscount,
		"l_tax":           outTax,
	})
}

func createOrder(c *gin.Context) {
	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		payload = map[string]interface{}{}
	}

	db := getDB()
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

	oOrderdate := payload["o_orderdate"]

	oComment := ""
	if v, ok := payload["o_comment"]; ok && v != nil {
		if s, ok := v.(string); ok {
			oComment = s
		}
	}

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
	oOrderkey := int(lastID)

	lineitemsRaw := payload["lineitems"].([]interface{})
	for _, itemRaw := range lineitemsRaw {
		item := itemRaw.(map[string]interface{})

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
			oOrderkey, item["l_linenumber"], item["l_extendedprice"],
			item["l_discount"], item["l_tax"], lShipdate, lComment,
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
	router.POST("/api/orders", createOrder)

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	if err := router.Run("0.0.0.0:" + port); err != nil {
		log.Fatal(err)
	}
}