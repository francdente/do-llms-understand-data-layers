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

Order.hasMany(Lineitem, { foreignKey: "l_orderkey", sourceKey: "o_orderkey" });
Lineitem.belongsTo(Order, { foreignKey: "l_orderkey", targetKey: "o_orderkey" });

app.get("/api/orders/:o_orderkey", async (req, res) => {
  const o_orderkey = parseInt(req.params.o_orderkey, 10);
  const rows = await sequelize.query(
    "SELECT o_orderkey, o_orderstatus, o_totalprice, o_orderdate, o_comment FROM orders WHERE o_orderkey = ?",
    {
      replacements: [o_orderkey],
      type: QueryTypes.SELECT,
    }
  );
  const row = rows[0];
  if (row == null) {
    return res.status(404).json({ error: "Order not found" });
  }
  return res.status(200).json(row);
});

app.put("/api/orders/:o_orderkey/lineitems/:l_linenumber", async (req, res) => {
  const o_orderkey = parseInt(req.params.o_orderkey, 10);
  const l_linenumber = parseInt(req.params.l_linenumber, 10);
  const payload = req.body || {};

  const transaction = await sequelize.transaction();
  try {
    const existingRows = await sequelize.query(
      "SELECT l_orderkey FROM lineitem WHERE l_orderkey = ? AND l_linenumber = ?",
      {
        replacements: [o_orderkey, l_linenumber],
        type: QueryTypes.SELECT,
        transaction,
      }
    );
    const existing = existingRows[0];

    if (existing == null) {
      await transaction.rollback();
      return res.status(404).json({ error: "Line item not found" });
    }

    await sequelize.query(
      "UPDATE lineitem SET l_extendedprice = ?, l_discount = ?, l_tax = ? WHERE l_orderkey = ? AND l_linenumber = ?",
      {
        replacements: [
          payload["l_extendedprice"],
          payload["l_discount"],
          payload["l_tax"],
          o_orderkey,
          l_linenumber,
        ],
        type: QueryTypes.UPDATE,
        transaction,
      }
    );

    const rows = await sequelize.query(
      "SELECT l_orderkey, l_linenumber, l_extendedprice, l_discount, l_tax FROM lineitem WHERE l_orderkey = ? AND l_linenumber = ?",
      {
        replacements: [o_orderkey, l_linenumber],
        type: QueryTypes.SELECT,
        transaction,
      }
    );
    const row = rows[0];

    await transaction.commit();
    return res.status(200).json(row);
  } catch (err) {
    if (!transaction.finished) {
      await transaction.rollback();
    }
    throw err;
  }
});

app.post("/api/orders", async (req, res) => {
  const payload = req.body || {};

  const transaction = await sequelize.transaction();
  try {
    const insertResult = await sequelize.query(
      "INSERT INTO orders (o_orderstatus, o_totalprice, o_orderdate, o_comment) VALUES (?, 0, ?, ?)",
      {
        replacements: [
          payload.o_orderstatus != null ? payload.o_orderstatus : "O",
          payload["o_orderdate"],
          payload.o_comment != null ? payload.o_comment : "",
        ],
        type: QueryTypes.INSERT,
        transaction,
      }
    );

    const o_orderkey = insertResult[0];

    for (const item of payload["lineitems"]) {
      await sequelize.query(
        "INSERT INTO lineitem (l_orderkey, l_linenumber, l_extendedprice, l_discount, l_tax, l_shipdate, l_comment) VALUES (?, ?, ?, ?, ?, ?, ?)",
        {
          replacements: [
            o_orderkey,
            item["l_linenumber"],
            item["l_extendedprice"],
            item["l_discount"],
            item["l_tax"],
            item["l_shipdate"] !== undefined ? item["l_shipdate"] : null,
            item["l_comment"] != null ? item["l_comment"] : "",
          ],
          type: QueryTypes.INSERT,
          transaction,
        }
      );
    }

    const totalRows = await sequelize.query(
      "SELECT COALESCE(SUM(l_extendedprice * (1 - l_discount) * (1 + l_tax)), 0) AS total FROM lineitem WHERE l_orderkey = ?",
      {
        replacements: [o_orderkey],
        type: QueryTypes.SELECT,
        transaction,
      }
    );
    const total = totalRows[0].total;

    await sequelize.query(
      "UPDATE orders SET o_totalprice = ? WHERE o_orderkey = ?",
      {
        replacements: [total, o_orderkey],
        type: QueryTypes.UPDATE,
        transaction,
      }
    );

    await transaction.commit();
    return res.status(201).json({ o_orderkey: o_orderkey, o_totalprice: total });
  } catch (err) {
    if (!transaction.finished) {
      await transaction.rollback();
    }
    throw err;
  }
});

(async () => {
  await sequelize.authenticate();
  app.listen(parseInt(process.env.PORT || "5000", 10), "0.0.0.0");
})();