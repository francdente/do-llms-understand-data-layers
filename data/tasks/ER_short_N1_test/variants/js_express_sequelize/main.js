const os = require("os");
const express = require("express");
const { Sequelize, DataTypes, QueryTypes } = require("sequelize");

const app = express();
const DATABASE = process.env.DB_PATH || "app.db";

const sequelize = new Sequelize({
  dialect: "sqlite",
  storage: DATABASE,
  logging: false,
});

const Product = sequelize.define(
  "product",
  {
    p_productkey: {
      type: DataTypes.INTEGER,
      primaryKey: true,
    },
    p_name: {
      type: DataTypes.TEXT,
    },
    p_status: {
      type: DataTypes.TEXT,
    },
    p_retailprice: {
      type: DataTypes.REAL,
    },
  },
  {
    tableName: "product",
    timestamps: false,
  }
);

app.use(express.json());

app.post("/api/products/:p_productkey/discontinue", async (req, res) => {
  const p_productkey = parseInt(req.params.p_productkey, 10);

  try {
    const existing = await sequelize.query(
      "SELECT p_productkey FROM product WHERE p_productkey = ?",
      {
        replacements: [p_productkey],
        type: QueryTypes.SELECT,
      }
    );

    if (existing.length === 0) {
      return res.status(404).json({ error: "Product not found" });
    }

    await sequelize.query(
      "UPDATE product SET p_status = 'discontinued' WHERE p_productkey = ?",
      {
        replacements: [p_productkey],
        type: QueryTypes.UPDATE,
      }
    );

    return res.status(200).json({
      p_productkey: p_productkey,
      p_status: "discontinued",
    });
  } catch (error) {
    return res.status(500).json({ error: "Internal Server Error" });
  }
});

app.get("/api/products", async (req, res) => {
  /**
   * List purchasable products
   */
  try {
    const rows = await sequelize.query(
      "SELECT p_productkey, p_name, p_retailprice " +
        "FROM product WHERE p_status = 'active' " +
        "ORDER BY p_productkey",
      {
        type: QueryTypes.SELECT,
      }
    );

    return res.status(200).json(rows);
  } catch (error) {
    return res.status(500).json({ error: "Internal Server Error" });
  }
});

async function start() {
  await sequelize.authenticate();
  app.listen(parseInt(process.env.PORT || "5000", 10), "0.0.0.0");
}

start();