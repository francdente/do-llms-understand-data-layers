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
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "app.db"
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	r := gin.Default()

	r.POST("/api/products/:p_productkey/archive", func(c *gin.Context) {
		pkStr := c.Param("p_productkey")
		pk, err := strconv.Atoi(pkStr)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}

		var existing struct {
			PProductkey int `gorm:"column:p_productkey"`
		}
		err = db.Table("product").
			Select("p_productkey").
			Where("p_productkey = ?", pk).
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
			Where("p_productkey = ?", pk).
			Update("p_status", "archived").Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"p_productkey": pk,
			"p_status":     "archived",
		})
	})

	r.GET("/api/products/archived", func(c *gin.Context) {
		var rows []struct {
			PProductkey  int      `gorm:"column:p_productkey" json:"p_productkey"`
			PName        *string  `gorm:"column:p_name" json:"p_name"`
			PRetailprice *float64 `gorm:"column:p_retailprice" json:"p_retailprice"`
		}

		if err := db.Table("product").
			Select("p_productkey, p_name, p_retailprice").
			Where("p_status = ?", "archived").
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