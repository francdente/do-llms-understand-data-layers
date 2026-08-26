const os = require("os");
const express = require("express");
const { Sequelize, DataTypes, Op } = require("sequelize");

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

async function get_db() {
  return sequelize;
}

async function close_db(exception) {
  return;
}

app.delete("/api/customers/:c_custkey", async (req, res) => {
  const c_custkey = parseInt(req.params.c_custkey, 10);
  const db = await get_db();
  const transaction = await db.transaction();
  try {
    const existing = await Customer.findOne({
      attributes: ["c_custkey"],
      where: { c_custkey },
      transaction,
    });
    if (existing === null) {
      await transaction.rollback();
      return res.status(404).json({ error: "Customer not found" });
    }
    await Order.destroy({
      where: { o_custkey: c_custkey },
      transaction,
    });
    await Customer.destroy({
      where: { c_custkey },
      transaction,
    });
    await transaction.commit();
    return res.status(204).send("");
  } catch (error) {
    await transaction.rollback();
    return res.status(500).json({ error: "Internal Server Error" });
  }
});

app.get("/api/orders", async (req, res) => {
  /**
   * Return orders tied to active customers
   */
  try {
    const db = await get_db();
    const rows = await Order.findAll({
      attributes: [
        "o_orderkey",
        "o_custkey",
        "o_totalprice",
        "o_orderdate",
        "o_comment",
      ],
      where: {
        o_custkey: {
          [Op.in]: Sequelize.literal("(SELECT c_custkey FROM customer)"),
        },
      },
      order: [["o_orderdate", "ASC"]],
      raw: true,
    });
    return res.status(200).json(rows);
  } catch (error) {
    return res.status(500).json({ error: "Internal Server Error" });
  }
});

async function start() {
  await sequelize.authenticate();
  const port = parseInt(process.env.PORT || "5000", 10);
  app.listen(port, "0.0.0.0");
}

start();