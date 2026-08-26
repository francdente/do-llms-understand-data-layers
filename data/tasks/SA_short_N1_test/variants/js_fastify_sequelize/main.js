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
      autoIncrement: true,
    },
    o_orderstatus: {
      type: DataTypes.TEXT,
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

app.get('/api/orders/:o_orderkey', async (request, reply) => {
  const o_orderkey = Number(request.params.o_orderkey);
  const rows = await sequelize.query(
    'SELECT o_orderkey, o_orderstatus, o_totalprice, o_orderdate, o_comment FROM orders WHERE o_orderkey = ?',
    {
      replacements: [o_orderkey],
      type: QueryTypes.SELECT,
    }
  );
  const row = rows[0];
  if (!row) {
    reply.code(404);
    return { error: 'Order not found' };
  }
  reply.code(200);
  return row;
});

app.put('/api/orders/:o_orderkey/lineitems/:l_linenumber', async (request, reply) => {
  const o_orderkey = Number(request.params.o_orderkey);
  const l_linenumber = Number(request.params.l_linenumber);
  const payload = request.body || {};
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
    if (!existing) {
      await transaction.rollback();
      reply.code(404);
      return { error: 'Line item not found' };
    }

    await sequelize.query(
      'UPDATE lineitem SET l_extendedprice = ?, l_discount = ?, l_tax = ? WHERE l_orderkey = ? AND l_linenumber = ?',
      {
        replacements: [
          payload.l_extendedprice,
          payload.l_discount,
          payload.l_tax,
          o_orderkey,
          l_linenumber,
        ],
        type: QueryTypes.UPDATE,
        transaction,
      }
    );

    await sequelize.query(
      'UPDATE orders SET o_totalprice = (SELECT COALESCE(SUM(l_extendedprice * (1 - l_discount) * (1 + l_tax)), 0)  FROM lineitem WHERE l_orderkey = ?) WHERE o_orderkey = ?',
      {
        replacements: [o_orderkey, o_orderkey],
        type: QueryTypes.UPDATE,
        transaction,
      }
    );

    const rows = await sequelize.query(
      'SELECT l_orderkey, l_linenumber, l_extendedprice, l_discount, l_tax FROM lineitem WHERE l_orderkey = ? AND l_linenumber = ?',
      {
        replacements: [o_orderkey, l_linenumber],
        type: QueryTypes.SELECT,
        transaction,
      }
    );
    const row = rows[0];
    await transaction.commit();
    reply.code(200);
    return row;
  } catch (err) {
    await transaction.rollback();
    throw err;
  }
});

app.post('/api/orders', async (request, reply) => {
  const payload = request.body || {};
  const transaction = await sequelize.transaction();
  try {
    const insertResult = await sequelize.query(
      'INSERT INTO orders (o_orderstatus, o_totalprice, o_orderdate, o_comment) VALUES (?, 0, ?, ?)',
      {
        replacements: [
          payload.o_orderstatus !== undefined ? payload.o_orderstatus : 'O',
          payload.o_orderdate,
          payload.o_comment !== undefined ? payload.o_comment : '',
        ],
        transaction,
      }
    );

    const o_orderkey = insertResult[0];

    for (const item of payload.lineitems) {
      await sequelize.query(
        'INSERT INTO lineitem (l_orderkey, l_linenumber, l_extendedprice, l_discount, l_tax, l_shipdate, l_comment) VALUES (?, ?, ?, ?, ?, ?, ?)',
        {
          replacements: [
            o_orderkey,
            item.l_linenumber,
            item.l_extendedprice,
            item.l_discount,
            item.l_tax,
            item.l_shipdate !== undefined ? item.l_shipdate : null,
            item.l_comment !== undefined ? item.l_comment : '',
          ],
          transaction,
        }
      );
    }

    const totalRows = await sequelize.query(
      'SELECT COALESCE(SUM(l_extendedprice * (1 - l_discount) * (1 + l_tax)), 0) AS total FROM lineitem WHERE l_orderkey = ?',
      {
        replacements: [o_orderkey],
        type: QueryTypes.SELECT,
        transaction,
      }
    );
    const total = totalRows[0].total;

    await sequelize.query(
      'UPDATE orders SET o_totalprice = ? WHERE o_orderkey = ?',
      {
        replacements: [total, o_orderkey],
        type: QueryTypes.UPDATE,
        transaction,
      }
    );

    await transaction.commit();
    reply.code(201);
    return { o_orderkey, o_totalprice: total };
  } catch (err) {
    await transaction.rollback();
    throw err;
  }
});

const start = async () => {
  await sequelize.authenticate();
  await app.listen({ host: '0.0.0.0', port: Number(process.env.PORT || 5000) });
};

start();