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

const Customer = sequelize.define(
  "customer",
  {
    c_custkey: {
      type: DataTypes.INTEGER,
      primaryKey: true,
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

    await sequelize.transaction(async (transaction) => {
      await Order.destroy({
        where: { o_custkey: c_custkey },
        transaction,
      });
      await Customer.destroy({
        where: { c_custkey },
        transaction,
      });
    });

    return res.status(204).send("");
  } catch (error) {
    return res.status(500).json({ error: "Internal Server Error" });
  }
});

app.get("/api/orders/monthly-trend", async (req, res) => {
  /**
   * Return the trend of all existed orders per-month.
   */
  try {
    const rows = await sequelize.query(
      "SELECT strftime('%Y-%m', o_orderdate) AS month, " +
        "       COUNT(*) AS order_count, " +
        "       SUM(o_totalprice) AS revenue " +
        "FROM orders " +
        "GROUP BY month " +
        "ORDER BY month",
      { type: QueryTypes.SELECT }
    );
    return res.status(200).json(rows);
  } catch (error) {
    return res.status(500).json({ error: "Internal Server Error" });
  }
});

if (require.main === module) {
  app.listen(parseInt(process.env.PORT || "5000", 10), "0.0.0.0");
}