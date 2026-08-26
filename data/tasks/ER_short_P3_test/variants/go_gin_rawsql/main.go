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

	router := gin.Default()

	router.POST("/api/suppliers/:s_suppkey/suspend", suspendSupplier)
	router.GET("/api/products", listProducts)

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	if err := router.Run("0.0.0.0:" + port); err != nil {
		panic(err)
	}
}

func suspendSupplier(c *gin.Context) {
	sSuppkey, err := strconv.Atoi(c.Param("s_suppkey"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Supplier not found"})
		return
	}

	var existing int
	err = db.QueryRow(
		"SELECT s_suppkey FROM supplier WHERE s_suppkey = ?",
		sSuppkey,
	).Scan(&existing)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Supplier not found"})
		return
	}
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	_, err = db.Exec(
		"UPDATE supplier SET s_status = 'suspended' WHERE s_suppkey = ?",
		sSuppkey,
	)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, gin.H{"s_suppkey": sSuppkey, "s_status": "suspended"})
}

func listProducts(c *gin.Context) {
	/**
    List purchasable products
    */
	rows, err := db.Query(
		"SELECT p.p_productkey, p.p_name, p.p_retailprice, s.s_name AS supplier_name " +
			"FROM product p JOIN supplier s ON p.p_suppkey = s.s_suppkey " +
			"ORDER BY p.p_productkey",
	)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	result := make([]gin.H, 0)
	for rows.Next() {
		var pProductkey int
		var pName sql.NullString
		var pRetailprice sql.NullFloat64
		var supplierName sql.NullString

		if err := rows.Scan(&pProductkey, &pName, &pRetailprice, &supplierName); err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		item := gin.H{
			"p_productkey": pProductkey,
			"p_name":       nil,
			"p_retailprice": nil,
			"supplier_name": nil,
		}
		if pName.Valid {
			item["p_name"] = pName.String
		}
		if pRetailprice.Valid {
			item["p_retailprice"] = pRetailprice.Float64
		}
		if supplierName.Valid {
			item["supplier_name"] = supplierName.String
		}

		result = append(result, item)
	}

	if err := rows.Err(); err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, result)
}