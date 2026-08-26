const os = require('os');
const Fastify = require('fastify');
const Database = require('better-sqlite3');

const app = Fastify();
const DATABASE = process.env.DB_PATH || 'app.db';
const db = new Database(DATABASE);

app.post('/api/suppliers/:s_suppkey/suspend', async (request, reply) => {
  const s_suppkey = Number(request.params.s_suppkey);
  const existing = db
    .prepare('SELECT s_suppkey FROM supplier WHERE s_suppkey = ?')
    .get(s_suppkey);

  if (existing == null) {
    reply.code(404);
    return { error: 'Supplier not found' };
  }

  db.prepare(
    "UPDATE supplier SET s_status = 'suspended' WHERE s_suppkey = ?"
  ).run(s_suppkey);

  reply.code(200);
  return { s_suppkey, s_status: 'suspended' };
});

app.get('/api/products', async (request, reply) => {
  /**
   * List purchasable products
   */
  const rows = db
    .prepare(
      'SELECT p.p_productkey, p.p_name, p.p_retailprice, s.s_name AS supplier_name ' +
        'FROM product p JOIN supplier s ON p.p_suppkey = s.s_suppkey ' +
        'ORDER BY p.p_productkey'
    )
    .all();

  reply.code(200);
  return rows;
});

const port = parseInt(process.env.PORT || '5000', 10);
app.listen({ host: '0.0.0.0', port }).catch((err) => {
  console.error(err);
  process.exit(1);
});