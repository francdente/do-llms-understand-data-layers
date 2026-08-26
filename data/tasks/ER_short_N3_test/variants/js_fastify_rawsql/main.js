const Fastify = require('fastify');
const Database = require('better-sqlite3');

const app = Fastify({ logger: false });
const DATABASE = process.env.DB_PATH || 'app.db';
const PORT = parseInt(process.env.PORT || '5000', 10);

const db = new Database(DATABASE);

app.post('/api/products/:p_productkey/archive', async (request, reply) => {
  const p_productkey = Number(request.params.p_productkey);

  const existing = db
    .prepare('SELECT p_productkey FROM product WHERE p_productkey = ?')
    .get(p_productkey);

  if (existing === undefined) {
    reply.code(404);
    return { error: 'Product not found' };
  }

  db.prepare(
    "UPDATE product SET p_status = 'archived' WHERE p_productkey = ?"
  ).run(p_productkey);

  reply.code(200);
  return { p_productkey, p_status: 'archived' };
});

app.get('/api/products/archived', async (request, reply) => {
  const rows = db
    .prepare(
      "SELECT p_productkey, p_name, p_retailprice " +
      "FROM product WHERE p_status = 'archived' " +
      "ORDER BY p_productkey"
    )
    .all();

  reply.code(200);
  return rows;
});

const start = async () => {
  try {
    await app.listen({ host: '0.0.0.0', port: PORT });
  } catch (err) {
    process.exit(1);
  }
};

const shutdown = () => {
  try {
    db.close();
  } catch (e) {}
  process.exit(0);
};

process.on('SIGINT', shutdown);
process.on('SIGTERM', shutdown);

start();