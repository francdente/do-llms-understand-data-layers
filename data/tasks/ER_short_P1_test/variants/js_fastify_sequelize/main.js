const os = require('os');
const Fastify = require('fastify');
const { Sequelize, DataTypes } = require('sequelize');

const app = Fastify({ logger: false });
const DATABASE = process.env.DB_PATH || 'app.db';

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
    },
    p_status: {
      type: DataTypes.TEXT,
    },
    p_retailprice: {
      type: DataTypes.REAL,
    },
  },
  {
    tableName: 'product',
    timestamps: false,
    freezeTableName: true,
  }
);

app.post('/api/products/:p_productkey/discontinue', async (request, reply) => {
  const p_productkey = parseInt(request.params.p_productkey, 10);

  const existing = await Product.findOne({
    attributes: ['p_productkey'],
    where: { p_productkey },
    raw: true,
  });

  if (existing === null) {
    reply.code(404);
    return { error: 'Product not found' };
  }

  await Product.update(
    { p_status: 'discontinued' },
    { where: { p_productkey } }
  );

  reply.code(200);
  return { p_productkey, p_status: 'discontinued' };
});

app.get('/api/products/:p_productkey', async (request, reply) => {
  /**
   * Return purchasable product
   */
  const p_productkey = parseInt(request.params.p_productkey, 10);

  const row = await Product.findOne({
    attributes: ['p_productkey', 'p_name', 'p_status', 'p_retailprice'],
    where: { p_productkey },
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
    const port = parseInt(process.env.PORT || '5000', 10);
    await app.listen({ host: '0.0.0.0', port });
  } catch (err) {
    console.error(err);
    process.exit(1);
  }
};

const shutdown = async () => {
  try {
    await sequelize.close();
  } catch (err) {
  }
  process.exit(0);
};

process.on('SIGINT', shutdown);
process.on('SIGTERM', shutdown);

start();