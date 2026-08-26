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

const Customer = sequelize.define(
  'customer',
  {
    c_custkey: {
      type: DataTypes.INTEGER,
      primaryKey: true,
      allowNull: false,
    },
    c_name: {
      type: DataTypes.TEXT,
    },
    c_email: {
      type: DataTypes.TEXT,
    },
  },
  {
    tableName: 'customer',
    timestamps: false,
  }
);

const Order = sequelize.define(
  'orders',
  {
    o_orderkey: {
      type: DataTypes.INTEGER,
      primaryKey: true,
      allowNull: false,
    },
    o_custkey: {
      type: DataTypes.INTEGER,
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
    tableName: 'orders',
    timestamps: false,
  }
);

Order.belongsTo(Customer, { foreignKey: 'o_custkey', targetKey: 'c_custkey' });
Customer.hasMany(Order, { foreignKey: 'o_custkey', sourceKey: 'c_custkey' });

app.delete('/api/customers/:c_custkey', async (request, reply) => {
  const c_custkey = parseInt(request.params.c_custkey, 10);

  const existing = await Customer.findOne({
    attributes: ['c_custkey'],
    where: { c_custkey },
    raw: true,
  });

  if (existing === null) {
    return reply.code(404).send({ error: 'Customer not found' });
  }

  const transaction = await sequelize.transaction();
  try {
    await Order.destroy({
      where: { o_custkey: c_custkey },
      transaction,
    });

    await Customer.destroy({
      where: { c_custkey },
      transaction,
    });

    await transaction.commit();
    return reply.code(204).send();
  } catch (err) {
    await transaction.rollback();
    throw err;
  }
});

app.get('/api/orders/history', async (request, reply) => {
  /**
   * Return the history of all existed orders
   */
  const rows = await Order.findAll({
    attributes: ['o_orderkey', 'o_custkey', 'o_totalprice', 'o_orderdate', 'o_comment'],
    order: [['o_orderdate', 'ASC']],
    raw: true,
  });

  return reply.code(200).send(rows);
});

const start = async () => {
  try {
    await sequelize.authenticate();
    await app.listen({
      host: '0.0.0.0',
      port: parseInt(process.env.PORT || '5000', 10),
    });
  } catch (err) {
    console.error(err);
    process.exit(1);
  }
};

start();