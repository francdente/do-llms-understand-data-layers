package main

import (
	"database/sql"
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

type AvailabilityRow struct {
	PProductkey int            `json:"p_productkey"`
	PName       sql.NullString `json:"-"`
	TotalStock  int            `json:"total_stock"`
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

	sqlDB, err := db.DB()
	if err != nil {
		panic(err)
	}
	defer sqlDB.Close()

	app := gin.Default()

	app.POST("/api/products/:p_productkey/discontinue", func(c *gin.Context) {
		pProductkey, err := strconv.Atoi(c.Param("p_productkey"))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}

		var existing struct {
			PProductkey int `gorm:"column:p_productkey"`
		}
		result := db.Raw(
			"SELECT p_productkey FROM product WHERE p_productkey = ?",
			pProductkey,
		).Scan(&existing)
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
			return
		}
		if result.RowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}

		tx := db.Exec(
			"UPDATE product SET p_status = 'discontinued' WHERE p_productkey = ?",
			pProductkey,
		)
		if tx.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": tx.Error.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"p_productkey": pProductkey,
			"p_status":     "discontinued",
		})
	})

	app.GET("/api/products/:p_productkey/availability", func(c *gin.Context) {
		/**
		 * Get availability for purchasable product
		 */
		pProductkey, err := strconv.Atoi(c.Param("p_productkey"))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}

		type rawAvailabilityRow struct {
			PProductkey int            `gorm:"column:p_productkey"`
			PName       sql.NullString `gorm:"column:p_name"`
			TotalStock  int            `gorm:"column:total_stock"`
		}

		var row rawAvailabilityRow
		result := db.Raw(
			"SELECT p.p_productkey, p.p_name, "+
				"       COALESCE(SUM(i.i_quantity), 0) AS total_stock "+
				"FROM product p LEFT JOIN inventory i ON p.p_productkey = i.i_productkey "+
				"WHERE p.p_productkey = ? "+
				"GROUP BY p.p_productkey",
			pProductkey,
		).Scan(&row)
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
			return
		}
		if result.RowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}

		if row.PName.Valid {
			c.JSON(http.StatusOK, gin.H{
				"p_productkey": row.PProductkey,
				"p_name":       row.PName.String,
				"total_stock":  row.TotalStock,
			})
		} else {
			c.JSON(http.StatusOK, gin.H{
				"p_productkey": row.PProductkey,
				"p_name":       nil,
				"total_stock":  row.TotalStock,
			})
		}
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	if err := app.Run("0.0.0.0:" + port); err != nil {
		panic(err)
	}
}