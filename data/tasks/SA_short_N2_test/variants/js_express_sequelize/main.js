const os = require('os');
const express = require('express');
const { Sequelize, DataTypes, QueryTypes } = require('sequelize');

const app = express();
app.use(express.json());

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
    o_orderstatus: {
      type: DataTypes.TEXT,
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
      references: {
        model: 'orders',
        key: 'o_orderkey',
      },
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

app.get('/api/orders/:o_orderkey', async (req, res) => {
  const o_orderkey = parseInt(req.params.o_orderkey, 10);
  const transaction = await sequelize.transaction();
  try {
    const rows = await sequelize.query(
      'SELECT o_orderkey, o_orderstatus, o_orderdate, o_comment FROM orders WHERE o_orderkey = ?',
      {
        replacements: [o_orderkey],
        type: QueryTypes.SELECT,
        transaction,
      }
    );
    const row = rows[0];
    if (!row) {
      await transaction.rollback();
      return res.status(404).json({ error: 'Order not found' });
    }
    const totals = await sequelize.query(
      'SELECT COALESCE(SUM(l_extendedprice * (1 - l_discount) * (1 + l_tax)), 0) AS total FROM lineitem WHERE l_orderkey = ?',
      {
        replacements: [o_orderkey],
        type: QueryTypes.SELECT,
        transaction,
      }
    );
    await transaction.commit();
    const result = { ...row };
    result.o_totalprice = totals[0].total;
    return res.status(200).json(result);
  } catch (err) {
    if (!transaction.finished) {
      await transaction.rollback();
    }
    return res.status(500).json({ error: 'Internal Server Error' });
  }
});

app.put('/api/orders/:o_orderkey/lineitems/:l_linenumber', async (req, res) => {
  const o_orderkey = parseInt(req.params.o_orderkey, 10);
  const l_linenumber = parseInt(req.params.l_linenumber, 10);
  const payload = req.body || {};
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
      return res.status(404).json({ error: 'Line item not found' });
    }
    await sequelize.query(
      'UPDATE lineitem SET l_extendedprice = ?, l_discount = ?, l_tax = ? WHERE l_orderkey = ? AND l_linenumber = ?',
      {
        replacements: [
          payload['l_extendedprice'],
          payload['l_discount'],
          payload['l_tax'],
          o_orderkey,
          l_linenumber,
        ],
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
    await transaction.commit();
    return res.status(200).json(rows[0]);
  } catch (err) {
    if (!transaction.finished) {
      await transaction.rollback();
    }
    return res.status(500).json({ error: 'Internal Server Error' });
  }
});

(async () => {
  await sequelize.authenticate();
  app.listen(parseInt(process.env.PORT || '5000', 10), '0.0.0.0');
})();