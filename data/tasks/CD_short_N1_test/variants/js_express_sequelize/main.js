const os = require("os");
const express = require("express");
const { Sequelize, DataTypes } = require("sequelize");

const app = express();
const DATABASE = process.env.DB_PATH || "app.db";

const sequelize = new Sequelize({
  dialect: "sqlite",
  storage: DATABASE,
  logging: false,
});

const Customer = sequelize.define(
  "customer",
  {
    c_custkey: {
      type: DataTypes.INTEGER,
      primaryKey: true,
      allowNull: false,
    },
    c_name: {
      type: DataTypes.TEXT,
    },
    c_email: {
      type: DataTypes.TEXT,
    },
  },
  {
    tableName: "customer",
    timestamps: false,
  }
);

const Order = sequelize.define(
  "orders",
  {
    o_orderkey: {
      type: DataTypes.INTEGER,
      primaryKey: true,
      allowNull: false,
    },
    o_custkey: {
      type: DataTypes.INTEGER,
    },
    o_totalprice: {
      type: DataTypes.REAL,
    },
    o_orderdate: {
      type: DataTypes.TEXT,
    },
    o_comment: {
      type: DataTypes.TEXT,
    },
  },
  {
    tableName: "orders",
    timestamps: false,
  }
);

Order.belongsTo(Customer, { foreignKey: "o_custkey", targetKey: "c_custkey" });
Customer.hasMany(Order, { foreignKey: "o_custkey", sourceKey: "c_custkey" });

app.delete("/api/customers/:c_custkey", async (req, res) => {
  const c_custkey = parseInt(req.params.c_custkey, 10);

  try {
    const existing = await Customer.findOne({
      where: { c_custkey },
      attributes: ["c_custkey"],
      raw: true,
    });

    if (existing === null) {
      return res.status(404).json({ error: "Customer not found" });
    }

    await sequelize.transaction(async (t) => {
      await Order.destroy({
        where: { o_custkey: c_custkey },
        transaction: t,
      });
      await Customer.destroy({
        where: { c_custkey },
        transaction: t,
      });
    });

    return res.status(204).send("");
  } catch (err) {
    return res.status(500).json({ error: "Internal Server Error" });
  }
});

app.get("/api/orders", async (req, res) => {
  /**
   * Return orders tied to active customers
   */
  try {
    const rows = await Order.findAll({
      attributes: [
        "o_orderkey",
        "o_custkey",
        "o_totalprice",
        "o_orderdate",
        "o_comment",
      ],
      include: [
        {
          model: Customer,
          attributes: [],
          required: true,
        },
      ],
      order: [["o_orderdate", "ASC"]],
      raw: true,
    });

    return res.status(200).json(rows);
  } catch (err) {
    return res.status(500).json({ error: "Internal Server Error" });
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