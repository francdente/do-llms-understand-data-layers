const os = require("os");
const express = require("express");
const { Sequelize, DataTypes, QueryTypes } = require("sequelize");

const app = express();
app.use(express.json());

const DATABASE = process.env.DB_PATH || "app.db";

const sequelize = new Sequelize({
  dialect: "sqlite",
  storage: DATABASE,
  logging: false,
});

const Order = sequelize.define(
  "orders",
  {
    o_orderkey: {
      type: DataTypes.INTEGER,
      primaryKey: true,
      autoIncrement: true,
    },
    o_orderstatus: {
      type: DataTypes.TEXT,
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

const Lineitem = sequelize.define(
  "lineitem",
  {
    l_orderkey: {
      type: DataTypes.INTEGER,
      primaryKey: true,
      references: {
        model: Order,
        key: "o_orderkey",
      },
    },
    l_linenumber: {
      type: DataTypes.INTEGER,
      primaryKey: true,
    },
    l_extendedprice: {
      type: DataTypes.REAL,
    },
    l_discount: {
      type: DataTypes.REAL,
    },
    l_tax: {
      type: DataTypes.REAL,
    },
    l_shipdate: {
      type: DataTypes.TEXT,
    },
    l_comment: {
      type: DataTypes.TEXT,
    },
  },
  {
    tableName: "lineitem",
    timestamps: false,
  }
);

Order.hasMany(Lineitem, { foreignKey: "l_orderkey" });
Lineitem.belongsTo(Order, { foreignKey: "l_orderkey" });

app.get("/api/orders/:o_orderkey", async (req, res) => {
  const o_orderkey = parseInt(req.params.o_orderkey, 10);
  try {
    const row = await Order.findOne({
      attributes: [
        "o_orderkey",
        "o_orderstatus",
        "o_totalprice",
        "o_orderdate",
        "o_comment",
      ],
      where: { o_orderkey },
      raw: true,
    });
    if (row === null) {
      return res.status(404).json({ error: "Order not found" });
    }
    return res.status(200).json(row);
  } catch (err) {
    return res.status(500).json({ error: "Internal server error" });
  }
});

app.put("/api/orders/:o_orderkey/lineitems/:l_linenumber", async (req, res) => {
  const o_orderkey = parseInt(req.params.o_orderkey, 10);
  const l_linenumber = parseInt(req.params.l_linenumber, 10);
  const payload = req.body || {};

  const transaction = await sequelize.transaction();
  try {
    const existing = await Lineitem.findOne({
      attributes: ["l_orderkey"],
      where: { l_orderkey: o_orderkey, l_linenumber },
      transaction,
      raw: true,
    });

    if (existing === null) {
      await transaction.rollback();
      return res.status(404).json({ error: "Line item not found" });
    }

    await Lineitem.update(
      {
        l_extendedprice: payload["l_extendedprice"],
        l_discount: payload["l_discount"],
        l_tax: payload["l_tax"],
      },
      {
        where: { l_orderkey: o_orderkey, l_linenumber },
        transaction,
      }
    );

    await sequelize.query(
      "UPDATE orders SET o_totalprice = " +
        "(SELECT COALESCE(SUM(l_extendedprice * (1 - l_discount) * (1 + l_tax)), 0) " +
        " FROM lineitem WHERE l_orderkey = ?) " +
        "WHERE o_orderkey = ?",
      {
        replacements: [o_orderkey, o_orderkey],
        type: QueryTypes.UPDATE,
        transaction,
      }
    );

    const row = await Lineitem.findOne({
      attributes: [
        "l_orderkey",
        "l_linenumber",
        "l_extendedprice",
        "l_discount",
        "l_tax",
      ],
      where: { l_orderkey: o_orderkey, l_linenumber },
      transaction,
      raw: true,
    });

    await transaction.commit();
    return res.status(200).json(row);
  } catch (err) {
    await transaction.rollback();
    return res.status(500).json({ error: "Internal server error" });
  }
});

app.post("/api/orders", async (req, res) => {
  const payload = req.body || {};
  const transaction = await sequelize.transaction();

  try {
    const createdOrder = await Order.create(
      {
        o_orderstatus: payload.o_orderstatus !== undefined ? payload.o_orderstatus : "O",
        o_totalprice: 0,
        o_orderdate: payload["o_orderdate"],
        o_comment: payload.o_comment !== undefined ? payload.o_comment : "",
      },
      { transaction }
    );

    const o_orderkey = createdOrder.o_orderkey;

    for (const item of payload["lineitems"]) {
      await Lineitem.create(
        {
          l_orderkey: o_orderkey,
          l_linenumber: item["l_linenumber"],
          l_extendedprice: item["l_extendedprice"],
          l_discount: item["l_discount"],
          l_tax: item["l_tax"],
          l_shipdate: item["l_shipdate"],
          l_comment: item.l_comment !== undefined ? item.l_comment : "",
        },
        { transaction }
      );
    }

    const totalRow = await sequelize.query(
      "SELECT COALESCE(SUM(l_extendedprice * (1 - l_discount) * (1 + l_tax)), 0) AS total " +
        "FROM lineitem WHERE l_orderkey = ?",
      {
        replacements: [o_orderkey],
        type: QueryTypes.SELECT,
        transaction,
      }
    );
    const total = totalRow[0].total;

    await Order.update(
      { o_totalprice: total },
      {
        where: { o_orderkey },
        transaction,
      }
    );

    await transaction.commit();
    return res.status(201).json({ o_orderkey, o_totalprice: total });
  } catch (err) {
    await transaction.rollback();
    return res.status(500).json({ error: "Internal server error" });
  }
});

(async () => {
  await sequelize.authenticate();
  app.listen(parseInt(process.env.PORT || "5000", 10), "0.0.0.0");
})();