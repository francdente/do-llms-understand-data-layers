const os = require('os');
const path = require('path');
const Fastify = require('fastify');
const { Sequelize, DataTypes, QueryTypes } = require('sequelize');

const app = Fastify();
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
  }
);

const Inventory = sequelize.define(
  'inventory',
  {
    i_productkey: {
      type: DataTypes.INTEGER,
      primaryKey: true,
      references: {
        model: Product,
        key: 'p_productkey',
      },
    },
    i_warehouse: {
      type: DataTypes.TEXT,
      primaryKey: true,
    },
    i_quantity: {
      type: DataTypes.INTEGER,
    },
  },
  {
    tableName: 'inventory',
    timestamps: false,
  }
);

Product.hasMany(Inventory, { foreignKey: 'i_productkey' });
Inventory.belongsTo(Product, { foreignKey: 'i_productkey' });

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

app.get('/api/products/:p_productkey/availability', async (request, reply) => {
  /**
   * Get availability for purchasable product
   */
  const p_productkey = parseInt(request.params.p_productkey, 10);

  const rows = await sequelize.query(
    "SELECT p.p_productkey, p.p_name, " +
      "       COALESCE(SUM(i.i_quantity), 0) AS total_stock " +
      "FROM product p LEFT JOIN inventory i ON p.p_productkey = i.i_productkey " +
      "WHERE p.p_productkey = ? " +
      "GROUP BY p.p_productkey",
    {
      replacements: [p_productkey],
      type: QueryTypes.SELECT,
    }
  );

  const row = rows.length > 0 ? rows[0] : null;

  if (row === null) {
    reply.code(404);
    return { error: 'Product not found' };
  }

  reply.code(200);
  return row;
});

const start = async () => {
  await sequelize.authenticate();
  await app.listen({
    host: '0.0.0.0',
    port: parseInt(process.env.PORT || '5000', 10),
  });
};

start().catch(async (err) => {
  console.error(err);
  try {
    await sequelize.close();
  } catch (_) {}
  process.exit(1);
});

const shutdown = async () => {
  try {
    await app.close();
  } catch (_) {}
  try {
    await sequelize.close();
  } catch (_) {}
  process.exit(0);
};

process.on('SIGINT', shutdown);
process.on('SIGTERM', shutdown);