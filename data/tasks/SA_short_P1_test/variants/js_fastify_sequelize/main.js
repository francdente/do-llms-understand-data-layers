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
      references: {
        model: Order,
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

Order.hasMany(Lineitem, { foreignKey: 'l_orderkey' });
Lineitem.belongsTo(Order, { foreignKey: 'l_orderkey' });

app.get('/api/orders/:o_orderkey', async (request, reply) => {
  const o_orderkey = Number(request.params.o_orderkey);
  const row = await Order.findOne({
    attributes: ['o_orderkey', 'o_orderstatus', 'o_totalprice', 'o_orderdate', 'o_comment'],
    where: { o_orderkey },
    raw: true,
  });
  if (row === null) {
    return reply.code(404).send({ error: 'Order not found' });
  }
  return reply.code(200).send(row);
});

app.put('/api/orders/:o_orderkey/lineitems/:l_linenumber', async (request, reply) => {
  const o_orderkey = Number(request.params.o_orderkey);
  const l_linenumber = Number(request.params.l_linenumber);
  const payload = request.body || {};

  const transaction = await sequelize.transaction();
  try {
    const existing = await Lineitem.findOne({
      attributes: ['l_orderkey'],
      where: { l_orderkey: o_orderkey, l_linenumber },
      raw: true,
      transaction,
    });

    if (existing === null) {
      await transaction.rollback();
      return reply.code(404).send({ error: 'Line item not found' });
    }

    await Lineitem.update(
      {
        l_extendedprice: payload.l_extendedprice,
        l_discount: payload.l_discount,
        l_tax: payload.l_tax,
      },
      {
        where: { l_orderkey: o_orderkey, l_linenumber },
        transaction,
      }
    );

    const row = await Lineitem.findOne({
      attributes: ['l_orderkey', 'l_linenumber', 'l_extendedprice', 'l_discount', 'l_tax'],
      where: { l_orderkey: o_orderkey, l_linenumber },
      raw: true,
      transaction,
    });

    await transaction.commit();
    return reply.code(200).send(row);
  } catch (err) {
    await transaction.rollback();
    throw err;
  }
});

app.post('/api/orders', async (request, reply) => {
  const payload = request.body || {};
  const transaction = await sequelize.transaction();

  try {
    const createdOrder = await Order.create(
      {
        o_orderstatus: payload.o_orderstatus !== undefined ? payload.o_orderstatus : 'O',
        o_totalprice: 0,
        o_orderdate: payload.o_orderdate,
        o_comment: payload.o_comment !== undefined ? payload.o_comment : '',
      },
      { transaction }
    );

    const o_orderkey = createdOrder.o_orderkey;

    for (const item of payload.lineitems) {
      await Lineitem.create(
        {
          l_orderkey: o_orderkey,
          l_linenumber: item.l_linenumber,
          l_extendedprice: item.l_extendedprice,
          l_discount: item.l_discount,
          l_tax: item.l_tax,
          l_shipdate: item.l_shipdate !== undefined ? item.l_shipdate : null,
          l_comment: item.l_comment !== undefined ? item.l_comment : '',
        },
        { transaction }
      );
    }

    const totalRow = await sequelize.query(
      'SELECT COALESCE(SUM(l_extendedprice * (1 - l_discount) * (1 + l_tax)), 0) AS total FROM lineitem WHERE l_orderkey = ?',
      {
        replacements: [o_orderkey],
        type: QueryTypes.SELECT,
        plain: true,
        transaction,
      }
    );

    const total = totalRow.total;

    await Order.update(
      { o_totalprice: total },
      {
        where: { o_orderkey },
        transaction,
      }
    );

    await transaction.commit();
    return reply.code(201).send({ o_orderkey, o_totalprice: total });
  } catch (err) {
    await transaction.rollback();
    throw err;
  }
});

const start = async () => {
  await sequelize.authenticate();
  await app.listen({
    host: '0.0.0.0',
    port: Number(process.env.PORT || 5000),
  });
};

start().catch((err) => {
  console.error(err);
  process.exit(1);
});