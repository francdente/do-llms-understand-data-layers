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

func getDB() *sql.DB {
	return db
}

func discontinueProduct(c *gin.Context) {
	pProductkey, err := strconv.Atoi(c.Param("p_productkey"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	db := getDB()
	var existing int
	err = db.QueryRow(
		"SELECT p_productkey FROM product WHERE p_productkey = ?",
		pProductkey,
	).Scan(&existing)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	_, err = db.Exec(
		"UPDATE product SET p_status = 'discontinued' WHERE p_productkey = ?",
		pProductkey,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"p_productkey": pProductkey, "p_status": "discontinued"})
}

func getProductAvailability(c *gin.Context) {
	/**
	 * Get availability for purchasable product
	 */
	pProductkey, err := strconv.Atoi(c.Param("p_productkey"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	db := getDB()
	var row struct {
		PProductkey int     `json:"p_productkey"`
		PName       string  `json:"p_name"`
		TotalStock  float64 `json:"total_stock"`
	}

	err = db.QueryRow(
		"SELECT p.p_productkey, p.p_name, "+
			"       COALESCE(SUM(i.i_quantity), 0) AS total_stock "+
			"FROM product p LEFT JOIN inventory i ON p.p_productkey = i.i_productkey "+
			"WHERE p.p_productkey = ? "+
			"GROUP BY p.p_productkey",
		pProductkey,
	).Scan(&row.PProductkey, &row.PName, &row.TotalStock)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"p_productkey": row.PProductkey,
		"p_name":       row.PName,
		"total_stock":  row.TotalStock,
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
		panic(err)
	}
	defer db.Close()

	router := gin.Default()

	router.POST("/api/products/:p_productkey/discontinue", discontinueProduct)
	router.GET("/api/products/:p_productkey/availability", getProductAvailability)

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	if err := router.Run("0.0.0.0:" + port); err != nil {
		panic(err)
	}
}