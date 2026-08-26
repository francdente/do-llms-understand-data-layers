const express = require("express");
const { Sequelize, DataTypes } = require("sequelize");

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
      allowNull: false,
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
    const existing = await Product.findOne({
      attributes: ["p_productkey"],
      where: { p_productkey },
      raw: true,
    });

    if (existing === null) {
      return res.status(404).json({ error: "Product not found" });
    }

    await Product.update(
      { p_status: "discontinued" },
      { where: { p_productkey } }
    );

    return res.status(200).json({
      p_productkey,
      p_status: "discontinued",
    });
  } catch (error) {
    return res.status(500).json({ error: "Internal server error" });
  }
});

app.get("/api/products/:p_productkey", async (req, res) => {
  /**
   * Return purchasable product
   */
  const p_productkey = parseInt(req.params.p_productkey, 10);

  try {
    const row = await Product.findOne({
      attributes: ["p_productkey", "p_name", "p_status", "p_retailprice"],
      where: { p_productkey },
      raw: true,
    });

    if (row === null) {
      return res.status(404).json({ error: "Product not found" });
    }

    return res.status(200).json(row);
  } catch (error) {
    return res.status(500).json({ error: "Internal server error" });
  }
});

async function start() {
  await sequelize.authenticate();
  app.listen(parseInt(process.env.PORT || "5000", 10), "0.0.0.0");
}

start();