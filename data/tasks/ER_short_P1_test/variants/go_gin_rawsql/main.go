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

func closeDB() {
	if db != nil {
		_ = db.Close()
	}
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	_, err = db.Exec(
		"UPDATE product SET p_status = 'discontinued' WHERE p_productkey = ?",
		pProductkey,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"p_productkey": pProductkey, "p_status": "discontinued"})
}

func getProduct(c *gin.Context) {
	/**
	Return purchasable product
	*/
	pProductkey, err := strconv.Atoi(c.Param("p_productkey"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	db := getDB()
	var product struct {
		PProductkey  int
		PName        sql.NullString
		PStatus      sql.NullString
		PRetailprice sql.NullFloat64
	}

	err = db.QueryRow(
		"SELECT p_productkey, p_name, p_status, p_retailprice "+
			"FROM product WHERE p_productkey = ?",
		pProductkey,
	).Scan(&product.PProductkey, &product.PName, &product.PStatus, &product.PRetailprice)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := gin.H{
		"p_productkey":  product.PProductkey,
		"p_name":        nil,
		"p_status":      nil,
		"p_retailprice": nil,
	}
	if product.PName.Valid {
		resp["p_name"] = product.PName.String
	}
	if product.PStatus.Valid {
		resp["p_status"] = product.PStatus.String
	}
	if product.PRetailprice.Valid {
		resp["p_retailprice"] = product.PRetailprice.Float64
	}

	c.JSON(http.StatusOK, resp)
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
	defer closeDB()

	if err = db.Ping(); err != nil {
		panic(err)
	}

	app := gin.Default()

	app.POST("/api/products/:p_productkey/discontinue", discontinueProduct)
	app.GET("/api/products/:p_productkey", getProduct)

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	if err := app.Run("0.0.0.0:" + port); err != nil {
		panic(err)
	}
}