package main

import (
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Supplier struct {
	SSuppkey int    `gorm:"column:s_suppkey;primaryKey"`
	SName    string `gorm:"column:s_name"`
	SStatus  string `gorm:"column:s_status"`
}

func (Supplier) TableName() string {
	return "supplier"
}

type Product struct {
	PProductkey  int     `gorm:"column:p_productkey;primaryKey"`
	PName        string  `gorm:"column:p_name"`
	PSuppkey     int     `gorm:"column:p_suppkey"`
	PRetailprice float64 `gorm:"column:p_retailprice"`
}

func (Product) TableName() string {
	return "product"
}

type ProductListRow struct {
	PProductkey  int     `json:"p_productkey" gorm:"column:p_productkey"`
	PName        string  `json:"p_name" gorm:"column:p_name"`
	PRetailprice float64 `json:"p_retailprice" gorm:"column:p_retailprice"`
	SupplierName string  `json:"supplier_name" gorm:"column:supplier_name"`
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

	sqlDB, err := db.DB()
	if err != nil {
		panic(err)
	}
	defer sqlDB.Close()

	r := gin.Default()

	r.POST("/api/suppliers/:s_suppkey/suspend", func(c *gin.Context) {
		sSuppkey, err := strconv.Atoi(c.Param("s_suppkey"))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Supplier not found"})
			return
		}

		var existing struct {
			SSuppkey int `gorm:"column:s_suppkey"`
		}
		result := db.Table("supplier").
			Select("s_suppkey").
			Where("s_suppkey = ?", sSuppkey).
			Take(&existing)

		if result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Supplier not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		if err := db.Table("supplier").
			Where("s_suppkey = ?", sSuppkey).
			Update("s_status", "suspended").Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"s_suppkey": sSuppkey, "s_status": "suspended"})
	})

	r.GET("/api/products", func(c *gin.Context) {
		/**
		 * List purchasable products
		 */
		var rows []ProductListRow
		err := db.Table("product p").
			Select("p.p_productkey, p.p_name, p.p_retailprice, s.s_name AS supplier_name").
			Joins("JOIN supplier s ON p.p_suppkey = s.s_suppkey").
			Order("p.p_productkey").
			Scan(&rows).Error
		if err != nil {
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