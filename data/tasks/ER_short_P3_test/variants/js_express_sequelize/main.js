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

app.post('/api/suppliers/:s_suppkey/suspend', async (req, res) => {
  const s_suppkey = parseInt(req.params.s_suppkey, 10);

  const existing = await Supplier.findOne({
    where: { s_suppkey },
    attributes: ['s_suppkey'],
    raw: true,
  });

  if (existing === null) {
    return res.status(404).json({ error: 'Supplier not found' });
  }

  await Supplier.update(
    { s_status: 'suspended' },
    { where: { s_suppkey } }
  );

  return res.status(200).json({ s_suppkey, s_status: 'suspended' });
});

app.get('/api/products', async (req, res) => {
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

  return res.status(200).json(rows);
});

const port = parseInt(process.env.PORT || '5000', 10);

sequelize.authenticate().then(() => {
  app.listen(port, '0.0.0.0');
}).catch((err) => {
  console.error(err);
  process.exit(1);
});