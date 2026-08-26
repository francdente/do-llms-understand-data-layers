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

type Product struct {
	PProductkey  int64   `json:"p_productkey"`
	PName        *string `json:"p_name"`
	PStatus      *string `json:"p_status"`
	PRetailprice *float64 `json:"p_retailprice"`
}

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "app.db"
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	router := gin.Default()

	router.PUT("/api/products/:p_productkey/price", func(c *gin.Context) {
		pProductkey, err := strconv.ParseInt(c.Param("p_productkey"), 10, 64)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}

		var payload map[string]interface{}
		if err := c.ShouldBindJSON(&payload); err != nil {
			payload = map[string]interface{}{}
		}

		var existing int64
		err = db.QueryRow(
			"SELECT p_productkey FROM product WHERE p_productkey = ?",
			pProductkey,
		).Scan(&existing)
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		_, err = db.Exec(
			"UPDATE product SET p_retailprice = ? WHERE p_productkey = ?",
			payload["p_retailprice"], pProductkey,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		row := db.QueryRow(
			"SELECT p_productkey, p_name, p_status, p_retailprice FROM product WHERE p_productkey = ?",
			pProductkey,
		)

		var product Product
		err = row.Scan(&product.PProductkey, &product.PName, &product.PStatus, &product.PRetailprice)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, product)
	})

	router.GET("/api/products/:p_productkey", func(c *gin.Context) {
		pProductkey, err := strconv.ParseInt(c.Param("p_productkey"), 10, 64)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}

		row := db.QueryRow(
			"SELECT p_productkey, p_name, p_status, p_retailprice FROM product WHERE p_productkey = ?",
			pProductkey,
		)

		var product Product
		err = row.Scan(&product.PProductkey, &product.PName, &product.PStatus, &product.PRetailprice)
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, product)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	if err := router.Run("0.0.0.0:" + port); err != nil {
		log.Fatal(err)
	}
}