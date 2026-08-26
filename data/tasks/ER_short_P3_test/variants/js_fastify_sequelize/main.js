const os = require('os');
const Fastify = require('fastify');
const { Sequelize, DataTypes } = require('sequelize');

const app = Fastify();
const DATABASE = process.env.DB_PATH || 'app.db';

const sequelize = new Sequelize({
  dialect: 'sqlite',
  storage: DATABASE,
  logging: false,
});

const Supplier = sequelize.define(
  'supplier',
  {
    s_suppkey: {
      type: DataTypes.INTEGER,
      primaryKey: true,
      allowNull: false,
    },
    s_name: {
      type: DataTypes.TEXT,
    },
    s_status: {
      type: DataTypes.TEXT,
    },
  },
  {
    tableName: 'supplier',
    timestamps: false,
  }
);

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
    p_suppkey: {
      type: DataTypes.INTEGER,
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

Product.belongsTo(Supplier, {
  foreignKey: 'p_suppkey',
  targetKey: 's_suppkey',
});
Supplier.hasMany(Product, {
  foreignKey: 'p_suppkey',
  sourceKey: 's_suppkey',
});

app.post('/api/suppliers/:s_suppkey/suspend', async (request, reply) => {
  const s_suppkey = parseInt(request.params.s_suppkey, 10);

  const existing = await Supplier.findOne({
    where: { s_suppkey },
    attributes: ['s_suppkey'],
    raw: true,
  });

  if (existing === null) {
    return reply.code(404).send({ error: 'Supplier not found' });
  }

  await Supplier.update(
    { s_status: 'suspended' },
    { where: { s_suppkey } }
  );

  return reply.code(200).send({ s_suppkey, s_status: 'suspended' });
});

app.get('/api/products', async (request, reply) => {
  /**
   * List purchasable products
   */
  const rows = await Product.findAll({
    attributes: [
      'p_productkey',
      'p_name',
      'p_retailprice',
      [Sequelize.col('supplier.s_name'), 'supplier_name'],
    ],
    include: [
      {
        model: Supplier,
        attributes: [],
        required: true,
      },
    ],
    order: [['p_productkey', 'ASC']],
    raw: true,
  });

  return reply.code(200).send(rows);
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

start();