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

const Product = sequelize.define(
  "product",
  {
    p_productkey: {
      type: DataTypes.INTEGER,
      primaryKey: true,
      allowNull: false,
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
    freezeTableName: true,
  }
);

// Keep all the comments in the original file.

/**
 * List purchasable products
 */
app.get("/api/products", async (request, reply) => {
  const rows = await Product.findAll({
    attributes: ["p_productkey", "p_name", "p_retailprice"],
    where: { p_status: "active" },
    order: [["p_productkey", "ASC"]],
    raw: true,
  });
  return reply.code(200).send(rows);
});

app.post("/api/products/:p_productkey/discontinue", async (request, reply) => {
  const p_productkey = Number(request.params.p_productkey);

  const existing = await Product.findOne({
    attributes: ["p_productkey"],
    where: { p_productkey },
    raw: true,
  });

  if (existing === null) {
    return reply.code(404).send({ error: "Product not found" });
  }

  await Product.update(
    { p_status: "discontinued" },
    { where: { p_productkey } }
  );

  return reply.code(200).send({
    p_productkey,
    p_status: "discontinued",
  });
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