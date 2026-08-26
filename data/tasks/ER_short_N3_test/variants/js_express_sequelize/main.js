const express = require('express');
const { Sequelize, DataTypes } = require('sequelize');

const app = express();
const DATABASE = process.env.DB_PATH || 'app.db';
const PORT = parseInt(process.env.PORT || '5000', 10);

const sequelize = new Sequelize({
  dialect: 'sqlite',
  storage: DATABASE,
  logging: false,
});

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
    p_status: {
      type: DataTypes.TEXT,
    },
    p_retailprice: {
      type: DataTypes.REAL,
    },
  },
  {
    tableName: 'product',
    timestamps: false,
    freezeTableName: true,
  }
);

app.post('/api/products/:p_productkey/archive', async (req, res) => {
  const p_productkey = parseInt(req.params.p_productkey, 10);

  try {
    const existing = await Product.findOne({
      attributes: ['p_productkey'],
      where: { p_productkey },
      raw: true,
    });

    if (existing === null) {
      return res.status(404).json({ error: 'Product not found' });
    }

    await Product.update(
      { p_status: 'archived' },
      { where: { p_productkey } }
    );

    return res.status(200).json({ p_productkey, p_status: 'archived' });
  } catch (err) {
    return res.status(500).json({ error: 'Internal Server Error' });
  }
});

app.get('/api/products/archived', async (req, res) => {
  try {
    const rows = await Product.findAll({
      attributes: ['p_productkey', 'p_name', 'p_retailprice'],
      where: { p_status: 'archived' },
      order: [['p_productkey', 'ASC']],
      raw: true,
    });

    return res.status(200).json(rows);
  } catch (err) {
    return res.status(500).json({ error: 'Internal Server Error' });
  }
});

(async () => {
  try {
    await sequelize.authenticate();
    app.listen(PORT, '0.0.0.0');
  } catch (err) {
    process.exit(1);
  }
})();