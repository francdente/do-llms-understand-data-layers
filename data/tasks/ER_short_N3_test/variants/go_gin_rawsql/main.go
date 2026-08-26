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

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	router := gin.Default()

	router.POST("/api/products/:p_productkey/archive", func(c *gin.Context) {
		pkStr := c.Param("p_productkey")
		pk, err := strconv.Atoi(pkStr)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}

		var existing int
		err = db.QueryRow(
			"SELECT p_productkey FROM product WHERE p_productkey = ?",
			pk,
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
			"UPDATE product SET p_status = 'archived' WHERE p_productkey = ?",
			pk,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"p_productkey": pk,
			"p_status":     "archived",
		})
	})

	router.GET("/api/products/archived", func(c *gin.Context) {
		rows, err := db.Query(
			"SELECT p_productkey, p_name, p_retailprice " +
				"FROM product WHERE p_status = 'archived' " +
				"ORDER BY p_productkey",
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}
		defer rows.Close()

		type product struct {
			PProductKey  int     `json:"p_productkey"`
			PName        string  `json:"p_name"`
			PRetailPrice float64 `json:"p_retailprice"`
		}

		result := make([]product, 0)
		for rows.Next() {
			var p product
			if err := rows.Scan(&p.PProductKey, &p.PName, &p.PRetailPrice); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
				return
			}
			result = append(result, p)
		}
		if err := rows.Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		c.JSON(http.StatusOK, result)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	if err := router.Run("0.0.0.0:" + port); err != nil {
		log.Fatal(err)
	}
}