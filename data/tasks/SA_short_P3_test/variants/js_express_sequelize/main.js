const os = require('os');
const express = require('express');
const { Sequelize, DataTypes } = require('sequelize');

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
      allowNull: false,
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
      allowNull: false,
      references: {
        model: 'orders',
        key: 'o_orderkey',
      },
    },
    l_linenumber: {
      type: DataTypes.INTEGER,
      primaryKey: true,
      allowNull: false,
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

app.use(express.json());

function todayIsoDate() {
  return new Date().toISOString().slice(0, 10);
}

app.post('/api/orders/:o_orderkey/lineitems/:l_linenumber/ship', async (req, res) => {
  const o_orderkey = parseInt(req.params.o_orderkey, 10);
  const l_linenumber = parseInt(req.params.l_linenumber, 10);

  try {
    const existing = await Lineitem.findOne({
      attributes: ['l_orderkey'],
      where: {
        l_orderkey: o_orderkey,
        l_linenumber: l_linenumber,
      },
      raw: true,
    });

    if (existing === null) {
      return res.status(404).json({ error: 'Line item not found' });
    }

    const today = todayIsoDate();

    await Lineitem.update(
      { l_shipdate: today },
      {
        where: {
          l_orderkey: o_orderkey,
          l_linenumber: l_linenumber,
        },
      }
    );

    return res.status(200).json({
      l_orderkey: o_orderkey,
      l_linenumber: l_linenumber,
      l_shipdate: today,
    });
  } catch (err) {
    return res.status(500).json({ error: 'Internal server error' });
  }
});

app.get('/api/orders/:o_orderkey', async (req, res) => {
  const o_orderkey = parseInt(req.params.o_orderkey, 10);

  try {
    const row = await Order.findOne({
      attributes: ['o_orderkey', 'o_totalprice', 'o_orderdate', 'o_latest_shipdate', 'o_comment'],
      where: {
        o_orderkey: o_orderkey,
      },
      raw: true,
    });

    if (row === null) {
      return res.status(404).json({ error: 'Order not found' });
    }

    return res.status(200).json(row);
  } catch (err) {
    return res.status(500).json({ error: 'Internal server error' });
  }
});

async function start() {
  await sequelize.authenticate();
  app.listen(parseInt(process.env.PORT || '5000', 10), '0.0.0.0');
}

start();