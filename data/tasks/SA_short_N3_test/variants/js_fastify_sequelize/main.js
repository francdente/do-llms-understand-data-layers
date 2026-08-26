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

const Order = sequelize.define(
  'orders',
  {
    o_orderkey: {
      type: DataTypes.INTEGER,
      primaryKey: true,
    },
    o_totalprice: {
      type: DataTypes.REAL,
    },
    o_orderdate: {
      type: DataTypes.TEXT,
    },
    o_latest_shipdate: {
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

const Lineitem = sequelize.define(
  'lineitem',
  {
    l_orderkey: {
      type: DataTypes.INTEGER,
      primaryKey: true,
    },
    l_linenumber: {
      type: DataTypes.INTEGER,
      primaryKey: true,
    },
    l_extendedprice: {
      type: DataTypes.REAL,
    },
    l_discount: {
      type: DataTypes.REAL,
    },
    l_tax: {
      type: DataTypes.REAL,
    },
    l_shipdate: {
      type: DataTypes.TEXT,
    },
    l_comment: {
      type: DataTypes.TEXT,
    },
  },
  {
    tableName: 'lineitem',
    timestamps: false,
  }
);

app.post('/api/orders/:o_orderkey/lineitems/:l_linenumber/ship', async (request, reply) => {
  const o_orderkey = parseInt(request.params.o_orderkey, 10);
  const l_linenumber = parseInt(request.params.l_linenumber, 10);

  const transaction = await sequelize.transaction();
  try {
    const existingRows = await sequelize.query(
      'SELECT l_orderkey FROM lineitem WHERE l_orderkey = ? AND l_linenumber = ?',
      {
        replacements: [o_orderkey, l_linenumber],
        type: QueryTypes.SELECT,
        transaction,
      }
    );
    const existing = existingRows[0];

    if (existing == null) {
      await transaction.rollback();
      return reply.code(404).send({ error: 'Line item not found' });
    }

    const today = new Date().toISOString().slice(0, 10);

    await sequelize.query(
      'UPDATE lineitem SET l_shipdate = ? WHERE l_orderkey = ? AND l_linenumber = ?',
      {
        replacements: [today, o_orderkey, l_linenumber],
        type: QueryTypes.UPDATE,
        transaction,
      }
    );

    const latestRows = await sequelize.query(
      'SELECT MAX(l_shipdate) AS latest FROM lineitem WHERE l_orderkey = ?',
      {
        replacements: [o_orderkey],
        type: QueryTypes.SELECT,
        transaction,
      }
    );
    const latest = latestRows[0].latest;

    await sequelize.query(
      'UPDATE orders SET o_latest_shipdate = ? WHERE o_orderkey = ?',
      {
        replacements: [latest, o_orderkey],
        type: QueryTypes.UPDATE,
        transaction,
      }
    );

    await transaction.commit();
    return reply.code(200).send({
      l_orderkey: o_orderkey,
      l_linenumber: l_linenumber,
      l_shipdate: today,
    });
  } catch (err) {
    await transaction.rollback();
    throw err;
  }
});

app.get('/api/orders/:o_orderkey', async (request, reply) => {
  const o_orderkey = parseInt(request.params.o_orderkey, 10);

  const rows = await sequelize.query(
    'SELECT o_orderkey, o_totalprice, o_orderdate, o_latest_shipdate, o_comment FROM orders WHERE o_orderkey = ?',
    {
      replacements: [o_orderkey],
      type: QueryTypes.SELECT,
    }
  );
  const row = rows[0];

  if (row == null) {
    return reply.code(404).send({ error: 'Order not found' });
  }
  return reply.code(200).send(row);
});

const start = async () => {
  await sequelize.authenticate();
  const port = parseInt(process.env.PORT || '5000', 10);
  await app.listen({ host: '0.0.0.0', port });
};

start();