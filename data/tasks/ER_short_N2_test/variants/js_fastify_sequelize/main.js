const fastify = require('fastify')({ logger: false });
const { Sequelize, DataTypes } = require('sequelize');

const DATABASE = process.env.DB_PATH || 'app.db';
const PORT = parseInt(process.env.PORT || '5000', 10);

const sequelize = new Sequelize({
  dialect: 'sqlite',
  storage: DATABASE,
  logging: false,
});

const Product = sequelize.define(
  'product',
  {
    p_productkey: {
      type: DataTypes.INTEGER,
      primaryKey: true,
      allowNull: false,
    },
    p_name: {
      type: DataTypes.TEXT,
      allowNull: true,
    },
    p_status: {
      type: DataTypes.TEXT,
      allowNull: true,
    },
    p_retailprice: {
      type: DataTypes.REAL,
      allowNull: true,
    },
  },
  {
    tableName: 'product',
    timestamps: false,
  }
);

fastify.put('/api/products/:p_productkey/price', async (request, reply) => {
  const p_productkey = parseInt(request.params.p_productkey, 10);
  const payload = request.body || {};

  const existing = await Product.findOne({
    where: { p_productkey },
    attributes: ['p_productkey'],
  });

  if (existing === null) {
    reply.code(404);
    return { error: 'Product not found' };
  }

  await Product.update(
    { p_retailprice: payload.p_retailprice },
    { where: { p_productkey } }
  );

  const row = await Product.findOne({
    where: { p_productkey },
    attributes: ['p_productkey', 'p_name', 'p_status', 'p_retailprice'],
    raw: true,
  });

  reply.code(200);
  return row;
});

fastify.get('/api/products/:p_productkey', async (request, reply) => {
  const p_productkey = parseInt(request.params.p_productkey, 10);

  const row = await Product.findOne({
    where: { p_productkey },
    attributes: ['p_productkey', 'p_name', 'p_status', 'p_retailprice'],
    raw: true,
  });

  if (row === null) {
    reply.code(404);
    return { error: 'Product not found' };
  }

  reply.code(200);
  return row;
});

const start = async () => {
  try {
    await sequelize.authenticate();
    await fastify.listen({ host: '0.0.0.0', port: PORT });
  } catch (err) {
    console.error(err);
    process.exit(1);
  }
};

start();