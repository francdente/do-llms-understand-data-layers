const path = require('path');
const Fastify = require('fastify');
const { Sequelize, DataTypes, QueryTypes, ForeignKeyConstraintError } = require('sequelize');

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
    freezeTableName: true,
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
    freezeTableName: true,
  }
);

Order.belongsTo(Customer, {
  foreignKey: 'o_custkey',
  targetKey: 'c_custkey',
});
Customer.hasMany(Order, {
  foreignKey: 'o_custkey',
  sourceKey: 'c_custkey',
});

async function getDb() {
  await sequelize.query('PRAGMA foreign_keys = ON;');
  return sequelize;
}

app.addHook('onClose', async () => {
  await sequelize.close();
});

app.delete('/api/customers/:c_custkey', async (request, reply) => {
  const c_custkey = Number(request.params.c_custkey);
  const db = await getDb();
  const existing = await db.query(
    'SELECT c_custkey FROM customer WHERE c_custkey = ?',
    {
      replacements: [c_custkey],
      type: QueryTypes.SELECT,
    }
  );

  if (existing.length === 0) {
    reply.code(404);
    return { error: 'Customer not found' };
  }

  try {
    await Customer.destroy({
      where: { c_custkey },
    });
  } catch (err) {
    if (err instanceof ForeignKeyConstraintError || (err && err.name === 'SequelizeForeignKeyConstraintError')) {
      reply.code(409);
      return { error: 'Cannot delete customer with existing orders' };
    }
    throw err;
  }

  reply.code(204);
  return;
});

app.get('/api/dashboard', async (request, reply) => {
  /**
   * Return dashboard based on all existed orders
   */
  const db = await getDb();
  const rows = await db.query(
    'SELECT COUNT(*) AS total_orders, ' +
      '       COALESCE(SUM(o_totalprice), 0) AS total_revenue ' +
      'FROM orders',
    {
      type: QueryTypes.SELECT,
    }
  );
  reply.code(200);
  return rows[0];
});

async function start() {
  await sequelize.authenticate();
  await sequelize.query('PRAGMA foreign_keys = ON;');
  const port = parseInt(process.env.PORT || '5000', 10);
  await app.listen({ host: '0.0.0.0', port });
}

start().catch((err) => {
  console.error(err);
  process.exit(1);
});