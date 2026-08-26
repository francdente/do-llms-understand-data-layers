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
	PName        *string  `gorm:"column:p_name" json:"p_name,omitempty"`
	PStatus      *string  `gorm:"column:p_status" json:"p_status,omitempty"`
	PRetailprice *float64 `gorm:"column:p_retailprice" json:"p_retailprice,omitempty"`
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

		if err := db.Table("product").
			Where("p_productkey = ?", pProductkey).
			Update("p_status", "discontinued").Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"p_productkey": pProductkey,
			"p_status":     "discontinued",
		})
	})

	r.GET("/api/products", func(c *gin.Context) {
		/*
		   List purchasable products
		*/
		type ProductListRow struct {
			PProductkey  int      `gorm:"column:p_productkey" json:"p_productkey"`
			PName        *string  `gorm:"column:p_name" json:"p_name"`
			PRetailprice *float64 `gorm:"column:p_retailprice" json:"p_retailprice"`
		}

		var rows []ProductListRow
		if err := db.Table("product").
			Select("p_productkey, p_name, p_retailprice").
			Where("p_status = 'active'").
			Order("p_productkey").
			Find(&rows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		c.JSON(http.StatusOK, rows)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	if err := r.Run("0.0.0.0:" + port); err != nil {
		panic(err)
	}
}