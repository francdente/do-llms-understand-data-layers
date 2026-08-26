const path = require("path");
const Fastify = require("fastify");
const { Sequelize, DataTypes } = require("sequelize");

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
    },
    o_totalprice: {
      type: DataTypes.REAL,
      allowNull: true,
    },
    o_orderdate: {
      type: DataTypes.TEXT,
      allowNull: true,
    },
    o_latest_shipdate: {
      type: DataTypes.TEXT,
      allowNull: true,
    },
    o_comment: {
      type: DataTypes.TEXT,
      allowNull: true,
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
      allowNull: true,
    },
    l_discount: {
      type: DataTypes.REAL,
      allowNull: true,
    },
    l_tax: {
      type: DataTypes.REAL,
      allowNull: true,
    },
    l_shipdate: {
      type: DataTypes.TEXT,
      allowNull: true,
    },
    l_comment: {
      type: DataTypes.TEXT,
      allowNull: true,
    },
  },
  {
    tableName: "lineitem",
    timestamps: false,
  }
);

Lineitem.belongsTo(Order, { foreignKey: "l_orderkey", targetKey: "o_orderkey" });
Order.hasMany(Lineitem, { foreignKey: "l_orderkey", sourceKey: "o_orderkey" });

app.post("/api/orders/:o_orderkey/lineitems/:l_linenumber/ship", async (request, reply) => {
  const o_orderkey = parseInt(request.params.o_orderkey, 10);
  const l_linenumber = parseInt(request.params.l_linenumber, 10);

  const existing = await Lineitem.findOne({
    attributes: ["l_orderkey"],
    where: {
      l_orderkey: o_orderkey,
      l_linenumber: l_linenumber,
    },
    raw: true,
  });

  if (existing === null) {
    return reply.code(404).send({ error: "Line item not found" });
  }

  const today = new Date().toISOString().slice(0, 10);

  await Lineitem.update(
    { l_shipdate: today },
    {
      where: {
        l_orderkey: o_orderkey,
        l_linenumber: l_linenumber,
      },
    }
  );

  return reply.code(200).send({
    l_orderkey: o_orderkey,
    l_linenumber: l_linenumber,
    l_shipdate: today,
  });
});

app.get("/api/orders/:o_orderkey", async (request, reply) => {
  const o_orderkey = parseInt(request.params.o_orderkey, 10);

  const row = await Order.findOne({
    attributes: ["o_orderkey", "o_totalprice", "o_orderdate", "o_latest_shipdate", "o_comment"],
    where: {
      o_orderkey: o_orderkey,
    },
    raw: true,
  });

  if (row === null) {
    return reply.code(404).send({ error: "Order not found" });
  }

  return reply.code(200).send(row);
});

const start = async () => {
  try {
    await sequelize.authenticate();
    const port = parseInt(process.env.PORT || "5000", 10);
    await app.listen({ host: "0.0.0.0", port });
  } catch (err) {
    console.error(err);
    process.exit(1);
  }
};

start();