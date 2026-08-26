const fastify = require('fastify')({ logger: false });
const { Sequelize, DataTypes } = require('sequelize');

const DATABASE = process.env.DB_PATH || 'app.db';
const PORT = parseInt(process.env.PORT || '5000', 10);

const sequelize = new Sequelize({
  dialect: 'sqlite',
  storage: DATABASE,
  logging: false
});

const Product = sequelize.define(
  'product',
  {
    p_productkey: {
      type: DataTypes.INTEGER,
      primaryKey: true
    },
    p_name: {
      type: DataTypes.TEXT
    },
    p_status: {
      type: DataTypes.TEXT
    },
    p_retailprice: {
      type: DataTypes.REAL
    }
  },
  {
    tableName: 'product',
    timestamps: false
  }
);

fastify.post('/api/products/:p_productkey/archive', async (request, reply) => {
  const p_productkey = parseInt(request.params.p_productkey, 10);

  const existing = await Product.findOne({
    attributes: ['p_productkey'],
    where: { p_productkey }
  });

  if (existing === null) {
    reply.code(404);
    return { error: 'Product not found' };
  }

  await Product.update(
    { p_status: 'archived' },
    { where: { p_productkey } }
  );

  reply.code(200);
  return { p_productkey, p_status: 'archived' };
});

fastify.get('/api/products/archived', async (request, reply) => {
  const rows = await Product.findAll({
    attributes: ['p_productkey', 'p_name', 'p_retailprice'],
    where: { p_status: 'archived' },
    order: [['p_productkey', 'ASC']],
    raw: true
  });

  reply.code(200);
  return rows;
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

process.on('SIGINT', async () => {
  try {
    await sequelize.close();
  } finally {
    process.exit(0);
  }
});

process.on('SIGTERM', async () => {
  try {
    await sequelize.close();
  } finally {
    process.exit(0);
  }
});