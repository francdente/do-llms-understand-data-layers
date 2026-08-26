const os = require("os");
const express = require("express");
const { Sequelize, DataTypes, QueryTypes, ForeignKeyConstraintError } = require("sequelize");

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
      references: {
        model: "customer",
        key: "c_custkey",
      },
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

Order.belongsTo(Customer, { foreignKey: "o_custkey" });
Customer.hasMany(Order, { foreignKey: "o_custkey" });

async function initDb() {
  await sequelize.authenticate();
  await sequelize.query("PRAGMA foreign_keys = ON;");
}

app.delete("/api/customers/:c_custkey", async (req, res) => {
  const c_custkey = parseInt(req.params.c_custkey, 10);

  const existing = await Customer.findOne({
    where: { c_custkey },
    attributes: ["c_custkey"],
    raw: true,
  });

  if (existing === null) {
    return res.status(404).json({ error: "Customer not found" });
  }

  try {
    await Customer.destroy({
      where: { c_custkey },
    });
  } catch (err) {
    if (err instanceof ForeignKeyConstraintError || (err && err.name === "SequelizeForeignKeyConstraintError")) {
      return res.status(409).json({ error: "Cannot delete customer with existing orders" });
    }
    throw err;
  }

  return res.status(204).send("");
});

app.get("/api/dashboard", async (req, res) => {
  /**
   * Return dashboard based on all existed orders
   */
  const rows = await sequelize.query(
    "SELECT COUNT(*) AS total_orders, " +
      "       COALESCE(SUM(o_totalprice), 0) AS total_revenue " +
      "FROM orders",
    {
      type: QueryTypes.SELECT,
    }
  );

  return res.status(200).json(rows[0]);
});

initDb()
  .then(() => {
    app.listen(parseInt(process.env.PORT || "5000", 10), "0.0.0.0");
  })
  .catch((err) => {
    console.error(err);
    process.exit(1);
  });