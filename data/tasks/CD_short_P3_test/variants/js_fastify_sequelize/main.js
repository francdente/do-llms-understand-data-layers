const os = require('os');
const Fastify = require('fastify');
const { Sequelize, DataTypes, QueryTypes } = require('sequelize');

const app = Fastify({ logger: false });
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

  const existing = await sequelize.query(
    'SELECT c_custkey FROM customer WHERE c_custkey = ?',
    {
      replacements: [c_custkey],
      type: QueryTypes.SELECT,
    }
  );

  if (existing.length === 0) {
    return reply.code(404).send({ error: 'Customer not found' });
  }

  const transaction = await sequelize.transaction();
  try {
    await sequelize.query('DELETE FROM orders WHERE o_custkey = ?', {
      replacements: [c_custkey],
      type: QueryTypes.DELETE,
      transaction,
    });
    await sequelize.query('DELETE FROM customer WHERE c_custkey = ?', {
      replacements: [c_custkey],
      type: QueryTypes.DELETE,
      transaction,
    });
    await transaction.commit();
    return reply.code(204).send();
  } catch (err) {
    await transaction.rollback();
    throw err;
  }
});

app.get('/api/orders/monthly-trend', async (request, reply) => {
  /**
   * Return the trend of all existed orders per-month.
   */
  const rows = await sequelize.query(
    "SELECT strftime('%Y-%m', o_orderdate) AS month, " +
      '       COUNT(*) AS order_count, ' +
      '       SUM(o_totalprice) AS revenue ' +
      'FROM orders ' +
      'GROUP BY month ' +
      'ORDER BY month',
    {
      type: QueryTypes.SELECT,
    }
  );
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

const shutdown = async () => {
  try {
    await sequelize.close();
  } finally {
    process.exit(0);
  }
};

process.on('SIGINT', shutdown);
process.on('SIGTERM', shutdown);

start();