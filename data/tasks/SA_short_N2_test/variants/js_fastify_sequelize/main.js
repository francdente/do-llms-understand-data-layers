const os = require("os");
const Fastify = require("fastify");
const { Sequelize, DataTypes, QueryTypes } = require("sequelize");

const app = Fastify();
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
    },
    o_orderstatus: {
      type: DataTypes.TEXT,
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

app.get("/api/orders/:o_orderkey", async (request, reply) => {
  const o_orderkey = Number(request.params.o_orderkey);
  const transaction = await sequelize.transaction();
  try {
    const rows = await sequelize.query(
      "SELECT o_orderkey, o_orderstatus, o_orderdate, o_comment FROM orders WHERE o_orderkey = ?",
      {
        replacements: [o_orderkey],
        type: QueryTypes.SELECT,
        transaction,
      }
    );
    const row = rows[0];
    if (!row) {
      await transaction.rollback();
      return reply.code(404).send({ error: "Order not found" });
    }
    const totals = await sequelize.query(
      "SELECT COALESCE(SUM(l_extendedprice * (1 - l_discount) * (1 + l_tax)), 0) AS total FROM lineitem WHERE l_orderkey = ?",
      {
        replacements: [o_orderkey],
        type: QueryTypes.SELECT,
        transaction,
      }
    );
    await transaction.commit();
    const result = { ...row };
    result.o_totalprice = totals[0].total;
    return reply.code(200).send(result);
  } catch (err) {
    if (!transaction.finished) {
      await transaction.rollback();
    }
    throw err;
  }
});

app.put("/api/orders/:o_orderkey/lineitems/:l_linenumber", async (request, reply) => {
  const o_orderkey = Number(request.params.o_orderkey);
  const l_linenumber = Number(request.params.l_linenumber);
  const payload = request.body && typeof request.body === "object" ? request.body : {};
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
    if (!existing) {
      await transaction.rollback();
      return reply.code(404).send({ error: "Line item not found" });
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
    return reply.code(200).send(row);
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