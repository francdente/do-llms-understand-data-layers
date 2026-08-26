const path = require("path");
const Fastify = require("fastify");
const { Sequelize, DataTypes, QueryTypes } = require("sequelize");

const app = Fastify({ logger: false });
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
        model: "orders",
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

app.get("/api/orders/:o_orderkey", async (request, reply) => {
  const o_orderkey = Number(request.params.o_orderkey);
  const rows = await sequelize.query(
    "SELECT o_orderkey, o_orderstatus, o_totalprice, o_orderdate, o_comment FROM orders WHERE o_orderkey = ?",
    {
      replacements: [o_orderkey],
      type: QueryTypes.SELECT,
    }
  );
  const row = rows[0];
  if (!row) {
    reply.code(404);
    return { error: "Order not found" };
  }
  reply.code(200);
  return row;
});

app.post("/api/orders/:o_orderkey/lineitems", async (request, reply) => {
  const o_orderkey = Number(request.params.o_orderkey);
  const payload = request.body || {};

  const transaction = await sequelize.transaction();
  try {
    const orders = await sequelize.query(
      "SELECT o_orderkey FROM orders WHERE o_orderkey = ?",
      {
        replacements: [o_orderkey],
        type: QueryTypes.SELECT,
        transaction,
      }
    );
    const order = orders[0];

    if (!order) {
      await transaction.rollback();
      reply.code(404);
      return { error: "Order not found" };
    }

    await sequelize.query(
      "INSERT INTO lineitem (l_orderkey, l_linenumber, l_extendedprice, l_discount, l_tax, l_shipdate, l_comment) VALUES (?, ?, ?, ?, ?, ?, ?)",
      {
        replacements: [
          o_orderkey,
          payload["l_linenumber"],
          payload["l_extendedprice"],
          payload["l_discount"],
          payload["l_tax"],
          Object.prototype.hasOwnProperty.call(payload, "l_shipdate") ? payload["l_shipdate"] : null,
          Object.prototype.hasOwnProperty.call(payload, "l_comment") ? payload["l_comment"] : "",
        ],
        type: QueryTypes.INSERT,
        transaction,
      }
    );

    await transaction.commit();
    reply.code(201);
    return { l_orderkey: o_orderkey, l_linenumber: payload["l_linenumber"] };
  } catch (err) {
    if (!transaction.finished) {
      await transaction.rollback();
    }
    throw err;
  }
});

app.post("/api/orders", async (request, reply) => {
  const payload = request.body || {};

  const transaction = await sequelize.transaction();
  try {
    const created = await Order.create(
      {
        o_orderstatus: Object.prototype.hasOwnProperty.call(payload, "o_orderstatus") ? payload["o_orderstatus"] : "O",
        o_totalprice: 0,
        o_orderdate: payload["o_orderdate"],
        o_comment: Object.prototype.hasOwnProperty.call(payload, "o_comment") ? payload["o_comment"] : "",
      },
      { transaction }
    );

    const o_orderkey = created.o_orderkey;

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
            Object.prototype.hasOwnProperty.call(item, "l_shipdate") ? item["l_shipdate"] : null,
            Object.prototype.hasOwnProperty.call(item, "l_comment") ? item["l_comment"] : "",
          ],
          type: QueryTypes.INSERT,
          transaction,
        }
      );
    }

    const totals = await sequelize.query(
      "SELECT COALESCE(SUM(l_extendedprice * (1 - l_discount) * (1 + l_tax)), 0) AS total FROM lineitem WHERE l_orderkey = ?",
      {
        replacements: [o_orderkey],
        type: QueryTypes.SELECT,
        transaction,
      }
    );
    const total = totals[0].total;

    await sequelize.query(
      "UPDATE orders SET o_totalprice = ? WHERE o_orderkey = ?",
      {
        replacements: [total, o_orderkey],
        type: QueryTypes.UPDATE,
        transaction,
      }
    );

    await transaction.commit();
    reply.code(201);
    return { o_orderkey, o_totalprice: total };
  } catch (err) {
    if (!transaction.finished) {
      await transaction.rollback();
    }
    throw err;
  }
});

const start = async () => {
  await sequelize.authenticate();
  await app.listen({
    host: "0.0.0.0",
    port: Number(process.env.PORT || 5000),
  });
};

start();