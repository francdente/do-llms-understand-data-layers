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

type UpdatePricePayload struct {
	PRetailprice float64 `json:"p_retailprice"`
}

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "app.db"
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	r := gin.Default()

	r.PUT("/api/products/:p_productkey/price", func(c *gin.Context) {
		pProductkey, err := strconv.Atoi(c.Param("p_productkey"))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}

		var payload UpdatePricePayload
		_ = c.ShouldBindJSON(&payload)

		var existing struct {
			PProductkey int `gorm:"column:p_productkey"`
		}
		err = db.Table("product").
			Select("p_productkey").
			Where("p_productkey = ?", pProductkey).
			Take(&existing).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		if err := db.Table("product").
			Where("p_productkey = ?", pProductkey).
			Update("p_retailprice", payload.PRetailprice).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		var row Product
		err = db.Table("product").
			Select("p_productkey, p_name, p_status, p_retailprice").
			Where("p_productkey = ?", pProductkey).
			Take(&row).Error
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		c.JSON(http.StatusOK, row)
	})

	r.GET("/api/products/:p_productkey", func(c *gin.Context) {
		pProductkey, err := strconv.Atoi(c.Param("p_productkey"))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}

		var row Product
		err = db.Table("product").
			Select("p_productkey, p_name, p_status, p_retailprice").
			Where("p_productkey = ?", pProductkey).
			Take(&row).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		c.JSON(http.StatusOK, row)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	if err := r.Run("0.0.0.0:" + port); err != nil {
		panic(err)
	}
}