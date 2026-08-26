const express = require("express");
const { Sequelize, DataTypes } = require("sequelize");

const app = express();
app.use(express.json());

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
      allowNull: true,
    },
    p_status: {
      type: DataTypes.TEXT,
      allowNull: true,
    },
    p_retailprice: {
      type: DataTypes.REAL,
      allowNull: true,
    },
  },
  {
    tableName: "product",
    timestamps: false,
    freezeTableName: true,
  }
);

app.put("/api/products/:p_productkey/price", async (req, res) => {
  const p_productkey = parseInt(req.params.p_productkey, 10);
  const payload = req.body || {};

  try {
    const existing = await Product.findOne({
      where: { p_productkey },
      attributes: ["p_productkey"],
    });

    if (existing === null) {
      return res.status(404).json({ error: "Product not found" });
    }

    await Product.update(
      { p_retailprice: payload["p_retailprice"] },
      { where: { p_productkey } }
    );

    const row = await Product.findOne({
      where: { p_productkey },
      attributes: ["p_productkey", "p_name", "p_status", "p_retailprice"],
    });

    return res.status(200).json(row.get({ plain: true }));
  } catch (err) {
    return res.status(500).json({ error: "Internal Server Error" });
  }
});

app.get("/api/products/:p_productkey", async (req, res) => {
  const p_productkey = parseInt(req.params.p_productkey, 10);

  try {
    const row = await Product.findOne({
      where: { p_productkey },
      attributes: ["p_productkey", "p_name", "p_status", "p_retailprice"],
    });

    if (row === null) {
      return res.status(404).json({ error: "Product not found" });
    }

    return res.status(200).json(row.get({ plain: true }));
  } catch (err) {
    return res.status(500).json({ error: "Internal Server Error" });
  }
});

async function start() {
  try {
    await sequelize.authenticate();
    const port = parseInt(process.env.PORT || "5000", 10);
    app.listen(port, "0.0.0.0");
  } catch (err) {
    process.exit(1);
  }
}

start();