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

app.delete('/api/customers/:c_custkey', async (req, res) => {
  const c_custkey = parseInt(req.params.c_custkey, 10);
  const transaction = await sequelize.transaction();
  try {
    const existing = await Customer.findOne({
      attributes: ['c_custkey'],
      where: { c_custkey },
      transaction,
    });
    if (existing === null) {
      await transaction.rollback();
      return res.status(404).json({ error: 'Customer not found' });
    }
    await Order.destroy({
      where: { o_custkey: c_custkey },
      transaction,
    });
    await Customer.destroy({
      where: { c_custkey },
      transaction,
    });
    await transaction.commit();
    return res.status(204).send('');
  } catch (error) {
    await transaction.rollback();
    return res.status(500).json({ error: 'Internal Server Error' });
  }
});

app.get('/api/orders/summary', async (req, res) => {
  /**
   * Return the summary of all existed historical orders
   */
  try {
    const rows = await sequelize.query(
      'SELECT COUNT(*) AS total_orders, ' +
        '       COALESCE(SUM(o_totalprice), 0) AS total_revenue ' +
        'FROM orders',
      { type: QueryTypes.SELECT }
    );
    return res.status(200).json(rows[0]);
  } catch (error) {
    return res.status(500).json({ error: 'Internal Server Error' });
  }
});

const port = parseInt(process.env.PORT || '5000', 10);

sequelize.authenticate().then(() => {
  app.listen(port, '0.0.0.0');
}).catch(() => {
  process.exit(1);
});