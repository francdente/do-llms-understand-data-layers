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

const Inventory = sequelize.define(
  "inventory",
  {
    i_productkey: {
      type: DataTypes.INTEGER,
      primaryKey: true,
      references: {
        model: Product,
        key: "p_productkey",
      },
    },
    i_warehouse: {
      type: DataTypes.TEXT,
      primaryKey: true,
    },
    i_quantity: {
      type: DataTypes.INTEGER,
    },
  },
  {
    tableName: "inventory",
    timestamps: false,
  }
);

Product.hasMany(Inventory, {
  foreignKey: "i_productkey",
  sourceKey: "p_productkey",
});
Inventory.belongsTo(Product, {
  foreignKey: "i_productkey",
  targetKey: "p_productkey",
});

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

    return res
      .status(200)
      .json({ p_productkey: p_productkey, p_status: "discontinued" });
  } catch (error) {
    return res.status(500).json({ error: "Internal server error" });
  }
});

app.get("/api/products/:p_productkey/availability", async (req, res) => {
  /**
   * Get availability for purchasable product
   */
  const p_productkey = parseInt(req.params.p_productkey, 10);

  try {
    const rows = await sequelize.query(
      "SELECT p.p_productkey, p.p_name, " +
        "       COALESCE(SUM(i.i_quantity), 0) AS total_stock " +
        "FROM product p LEFT JOIN inventory i ON p.p_productkey = i.i_productkey " +
        "WHERE p.p_productkey = ? " +
        "GROUP BY p.p_productkey",
      {
        replacements: [p_productkey],
        type: QueryTypes.SELECT,
      }
    );

    if (rows.length === 0) {
      return res.status(404).json({ error: "Product not found" });
    }

    return res.status(200).json(rows[0]);
  } catch (error) {
    return res.status(500).json({ error: "Internal server error" });
  }
});

const port = parseInt(process.env.PORT || "5000", 10);

sequelize
  .authenticate()
  .then(() => {
    app.listen(port, "0.0.0.0");
  })
  .catch(() => {
    process.exit(1);
  });