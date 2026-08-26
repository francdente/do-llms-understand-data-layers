package main

import (
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Product struct {
	PProductkey  int      `gorm:"column:p_productkey;primaryKey" json:"p_productkey"`
	PName        *string  `gorm:"column:p_name" json:"p_name"`
	PStatus      *string  `gorm:"column:p_status" json:"p_status"`
	PRetailprice *float64 `gorm:"column:p_retailprice" json:"p_retailprice"`
}

func (Product) TableName() string {
	return "product"
}

func main() {
	database := os.Getenv("DB_PATH")
	if database == "" {
		database = "app.db"
	}

	db, err := gorm.Open(sqlite.Open(database), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	r := gin.Default()

	r.POST("/api/products/:p_productkey/discontinue", func(c *gin.Context) {
		pProductkey, err := strconv.Atoi(c.Param("p_productkey"))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}

		var existing struct {
			PProductkey int `gorm:"column:p_productkey"`
		}
		result := db.Table("product").
			Select("p_productkey").
			Where("p_productkey = ?", pProductkey).
			Take(&existing)

		if result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		updateResult := db.Table("product").
			Where("p_productkey = ?", pProductkey).
			Update("p_status", "discontinued")
		if updateResult.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"p_productkey": pProductkey,
			"p_status":     "discontinued",
		})
	})

	r.GET("/api/products/:p_productkey", func(c *gin.Context) {
		/**
		Return purchasable product
		*/
		pProductkey, err := strconv.Atoi(c.Param("p_productkey"))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}

		var row Product
		result := db.Table("product").
			Select("p_productkey, p_name, p_status, p_retailprice").
			Where("p_productkey = ?", pProductkey).
			Take(&row)

		if result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"p_productkey":  row.PProductkey,
			"p_name":        row.PName,
			"p_status":      row.PStatus,
			"p_retailprice": row.PRetailprice,
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	if err := r.Run("0.0.0.0:" + port); err != nil {
		panic(err)
	}
}