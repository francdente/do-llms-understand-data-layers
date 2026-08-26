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

	app := gin.Default()

	app.POST("/api/products/:p_productkey/discontinue", discontinueProduct)
	app.GET("/api/products", listProducts)

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	if err := app.Run("0.0.0.0:" + port); err != nil {
		panic(err)
	}
}

func discontinueProduct(c *gin.Context) {
	pProductkey, err := strconv.Atoi(c.Param("p_productkey"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

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
		c.Status(http.StatusInternalServerError)
		return
	}

	_, err = db.Exec(
		"UPDATE product SET p_status = 'discontinued' WHERE p_productkey = ?",
		pProductkey,
	)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, gin.H{"p_productkey": pProductkey, "p_status": "discontinued"})
}

func listProducts(c *gin.Context) {
	/**
	List purchasable products
	*/
	rows, err := db.Query(
		"SELECT p_productkey, p_name, p_retailprice " +
			"FROM product WHERE p_status = 'active' " +
			"ORDER BY p_productkey",
	)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	result := make([]gin.H, 0)
	for rows.Next() {
		var pProductkey int
		var pName sql.NullString
		var pRetailprice sql.NullFloat64

		if err := rows.Scan(&pProductkey, &pName, &pRetailprice); err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		item := gin.H{
			"p_productkey": pProductkey,
			"p_name":       nil,
			"p_retailprice": nil,
		}
		if pName.Valid {
			item["p_name"] = pName.String
		}
		if pRetailprice.Valid {
			item["p_retailprice"] = pRetailprice.Float64
		}

		result = append(result, item)
	}

	if err := rows.Err(); err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, result)
}