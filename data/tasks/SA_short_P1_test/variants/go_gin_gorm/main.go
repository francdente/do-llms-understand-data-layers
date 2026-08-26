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

type Order struct {
	OOrderkey    int     `gorm:"column:o_orderkey;primaryKey;autoIncrement" json:"o_orderkey"`
	OOrderstatus string  `gorm:"column:o_orderstatus" json:"o_orderstatus"`
	OTotalprice  float64 `gorm:"column:o_totalprice" json:"o_totalprice"`
	OOrderdate   string  `gorm:"column:o_orderdate" json:"o_orderdate"`
	OComment     string  `gorm:"column:o_comment" json:"o_comment"`
}

func (Order) TableName() string {
	return "orders"
}

type Lineitem struct {
	LOrderkey      int      `gorm:"column:l_orderkey;primaryKey" json:"l_orderkey"`
	LLinenumber    int      `gorm:"column:l_linenumber;primaryKey" json:"l_linenumber"`
	LExtendedprice float64  `gorm:"column:l_extendedprice" json:"l_extendedprice"`
	LDiscount      float64  `gorm:"column:l_discount" json:"l_discount"`
	LTax           float64  `gorm:"column:l_tax" json:"l_tax"`
	LShipdate      *string  `gorm:"column:l_shipdate" json:"l_shipdate"`
	LComment       string   `gorm:"column:l_comment" json:"l_comment"`
}

func (Lineitem) TableName() string {
	return "lineitem"
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

	r.GET("/api/orders/:o_orderkey", func(c *gin.Context) {
		oOrderkey, err := strconv.Atoi(c.Param("o_orderkey"))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
			return
		}

		var row struct {
			OOrderkey    int     `json:"o_orderkey"`
			OOrderstatus string  `json:"o_orderstatus"`
			OTotalprice  float64 `json:"o_totalprice"`
			OOrderdate   string  `json:"o_orderdate"`
			OComment     string  `json:"o_comment"`
		}

		res := db.Raw(
			"SELECT o_orderkey, o_orderstatus, o_totalprice, o_orderdate, o_comment FROM orders WHERE o_orderkey = ?",
			oOrderkey,
		).Scan(&row)

		if res.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": res.Error.Error()})
			return
		}
		if res.RowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
			return
		}

		c.JSON(http.StatusOK, row)
	})

	r.PUT("/api/orders/:o_orderkey/lineitems/:l_linenumber", func(c *gin.Context) {
		oOrderkey, err1 := strconv.Atoi(c.Param("o_orderkey"))
		lLinenumber, err2 := strconv.Atoi(c.Param("l_linenumber"))
		if err1 != nil || err2 != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Line item not found"})
			return
		}

		var payload map[string]interface{}
		if err := c.ShouldBindJSON(&payload); err != nil {
			payload = map[string]interface{}{}
		}

		tx := db.Begin()
		if tx.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": tx.Error.Error()})
			return
		}

		var existing struct {
			LOrderkey int `json:"l_orderkey"`
		}
		res := tx.Raw(
			"SELECT l_orderkey FROM lineitem WHERE l_orderkey = ? AND l_linenumber = ?",
			oOrderkey, lLinenumber,
		).Scan(&existing)
		if res.Error != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": res.Error.Error()})
			return
		}
		if res.RowsAffected == 0 {
			tx.Rollback()
			c.JSON(http.StatusNotFound, gin.H{"error": "Line item not found"})
			return
		}

		_, ok1 := payload["l_extendedprice"]
		_, ok2 := payload["l_discount"]
		_, ok3 := payload["l_tax"]
		if !ok1 || !ok2 || !ok3 {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "missing required fields"})
			return
		}

		res = tx.Exec(
			"UPDATE lineitem SET l_extendedprice = ?, l_discount = ?, l_tax = ? WHERE l_orderkey = ? AND l_linenumber = ?",
			payload["l_extendedprice"], payload["l_discount"], payload["l_tax"], oOrderkey, lLinenumber,
		)
		if res.Error != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": res.Error.Error()})
			return
		}

		var row struct {
			LOrderkey      int     `json:"l_orderkey"`
			LLinenumber    int     `json:"l_linenumber"`
			LExtendedprice float64 `json:"l_extendedprice"`
			LDiscount      float64 `json:"l_discount"`
			LTax           float64 `json:"l_tax"`
		}
		res = tx.Raw(
			"SELECT l_orderkey, l_linenumber, l_extendedprice, l_discount, l_tax FROM lineitem WHERE l_orderkey = ? AND l_linenumber = ?",
			oOrderkey, lLinenumber,
		).Scan(&row)
		if res.Error != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": res.Error.Error()})
			return
		}

		if err := tx.Commit().Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, row)
	})

	r.POST("/api/orders", func(c *gin.Context) {
		var payload map[string]interface{}
		if err := c.ShouldBindJSON(&payload); err != nil {
			payload = map[string]interface{}{}
		}

		tx := db.Begin()
		if tx.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": tx.Error.Error()})
			return
		}

		oOrderstatus := "O"
		if v, ok := payload["o_orderstatus"]; ok && v != nil {
			if s, ok := v.(string); ok {
				oOrderstatus = s
			}
		}

		oComment := ""
		if v, ok := payload["o_comment"]; ok && v != nil {
			if s, ok := v.(string); ok {
				oComment = s
			}
		}

		oOrderdateVal, ok := payload["o_orderdate"]
		if !ok {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "missing required fields"})
			return
		}
		oOrderdate, ok := oOrderdateVal.(string)
		if !ok {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "missing required fields"})
			return
		}

		res := tx.Exec(
			"INSERT INTO orders (o_orderstatus, o_totalprice, o_orderdate, o_comment) VALUES (?, 0, ?, ?)",
			oOrderstatus, oOrderdate, oComment,
		)
		if res.Error != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": res.Error.Error()})
			return
		}

		var oOrderkey int64
		row := tx.Raw("SELECT last_insert_rowid()").Row()
		if err := row.Scan(&oOrderkey); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		lineitemsVal, ok := payload["lineitems"]
		if !ok {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "missing required fields"})
			return
		}
		lineitems, ok := lineitemsVal.([]interface{})
		if !ok {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "missing required fields"})
			return
		}

		for _, itemVal := range lineitems {
			item, ok := itemVal.(map[string]interface{})
			if !ok {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid lineitem"})
				return
			}

			lLinenumber, ok := toInt(item["l_linenumber"])
			if !ok {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid lineitem"})
				return
			}
			lExtendedprice, ok := toFloat(item["l_extendedprice"])
			if !ok {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid lineitem"})
				return
			}
			lDiscount, ok := toFloat(item["l_discount"])
			if !ok {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid lineitem"})
				return
			}
			lTax, ok := toFloat(item["l_tax"])
			if !ok {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid lineitem"})
				return
			}

			var lShipdate interface{} = nil
			if v, exists := item["l_shipdate"]; exists {
				switch t := v.(type) {
				case string:
					lShipdate = t
				case nil:
					lShipdate = nil
				default:
					lShipdate = nil
				}
			}

			lComment := ""
			if v, exists := item["l_comment"]; exists && v != nil {
				if s, ok := v.(string); ok {
					lComment = s
				}
			}

			res = tx.Exec(
				"INSERT INTO lineitem (l_orderkey, l_linenumber, l_extendedprice, l_discount, l_tax, l_shipdate, l_comment) VALUES (?, ?, ?, ?, ?, ?, ?)",
				oOrderkey, lLinenumber, lExtendedprice, lDiscount, lTax, lShipdate, lComment,
			)
			if res.Error != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": res.Error.Error()})
				return
			}
		}

		var total sql.NullFloat64
		row = tx.Raw(
			"SELECT COALESCE(SUM(l_extendedprice * (1 - l_discount) * (1 + l_tax)), 0) AS total FROM lineitem WHERE l_orderkey = ?",
			oOrderkey,
		).Row()
		if err := row.Scan(&total); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		totalValue := 0.0
		if total.Valid {
			totalValue = total.Float64
		}

		res = tx.Exec(
			"UPDATE orders SET o_totalprice = ? WHERE o_orderkey = ?",
			totalValue, oOrderkey,
		)
		if res.Error != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": res.Error.Error()})
			return
		}

		if err := tx.Commit().Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"o_orderkey":   oOrderkey,
			"o_totalprice": totalValue,
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}
	r.Run("0.0.0.0:" + port)
}

func toInt(v interface{}) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	case int64:
		return int(n), true
	default:
		return 0, false
	}
}

func toFloat(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}