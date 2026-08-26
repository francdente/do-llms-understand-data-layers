const os = require('os');
const express = require('express');
const { Sequelize, DataTypes, QueryTypes } = require('sequelize');

const app = express();
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

Order.hasMany(Lineitem, { foreignKey: 'l_orderkey', sourceKey: 'o_orderkey' });
Lineitem.belongsTo(Order, { foreignKey: 'l_orderkey', targetKey: 'o_orderkey' });

function todayIsoDate() {
  return new Date().toISOString().slice(0, 10);
}

app.post('/api/orders/:o_orderkey/lineitems/:l_linenumber/ship', async (req, res) => {
  const o_orderkey = parseInt(req.params.o_orderkey, 10);
  const l_linenumber = parseInt(req.params.l_linenumber, 10);

  try {
    const result = await sequelize.transaction(async (t) => {
      const existing = await sequelize.query(
        'SELECT l_orderkey FROM lineitem WHERE l_orderkey = ? AND l_linenumber = ?',
        {
          replacements: [o_orderkey, l_linenumber],
          type: QueryTypes.SELECT,
          transaction: t,
        }
      );

      if (existing.length === 0) {
        return { notFound: true };
      }

      const today = todayIsoDate();

      await sequelize.query(
        'UPDATE lineitem SET l_shipdate = ? WHERE l_orderkey = ? AND l_linenumber = ?',
        {
          replacements: [today, o_orderkey, l_linenumber],
          type: QueryTypes.UPDATE,
          transaction: t,
        }
      );

      const latestRows = await sequelize.query(
        'SELECT MAX(l_shipdate) AS latest FROM lineitem WHERE l_orderkey = ?',
        {
          replacements: [o_orderkey],
          type: QueryTypes.SELECT,
          transaction: t,
        }
      );

      const latest = latestRows[0] ? latestRows[0].latest : null;

      await sequelize.query(
        'UPDATE orders SET o_latest_shipdate = ? WHERE o_orderkey = ?',
        {
          replacements: [latest, o_orderkey],
          type: QueryTypes.UPDATE,
          transaction: t,
        }
      );

      return {
        notFound: false,
        body: {
          l_orderkey: o_orderkey,
          l_linenumber: l_linenumber,
          l_shipdate: today,
        },
      };
    });

    if (result.notFound) {
      return res.status(404).json({ error: 'Line item not found' });
    }

    return res.status(200).json(result.body);
  } catch (err) {
    return res.status(500).json({ error: 'Internal Server Error' });
  }
});

app.get('/api/orders/:o_orderkey', async (req, res) => {
  const o_orderkey = parseInt(req.params.o_orderkey, 10);

  try {
    const rows = await sequelize.query(
      'SELECT o_orderkey, o_totalprice, o_orderdate, o_latest_shipdate, o_comment FROM orders WHERE o_orderkey = ?',
      {
        replacements: [o_orderkey],
        type: QueryTypes.SELECT,
      }
    );

    if (rows.length === 0) {
      return res.status(404).json({ error: 'Order not found' });
    }

    return res.status(200).json(rows[0]);
  } catch (err) {
    return res.status(500).json({ error: 'Internal Server Error' });
  }
});

(async () => {
  try {
    await sequelize.authenticate();
    app.listen(parseInt(process.env.PORT || '5000', 10), '0.0.0.0');
  } catch (err) {
    process.exit(1);
  }
})();