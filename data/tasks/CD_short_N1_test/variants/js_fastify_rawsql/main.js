const Fastify = require('fastify');
const Database = require('better-sqlite3');

const app = Fastify();
const DATABASE = process.env.DB_PATH || 'app.db';
const db = new Database(DATABASE);

function get_db() {
    return db;
}

async function close_db() {
    const database = get_db();
    if (database) {
        database.close();
    }
}

app.addHook('onClose', async () => {
    await close_db();
});

app.delete('/api/customers/:c_custkey', async (request, reply) => {
    const c_custkey = Number(request.params.c_custkey);
    const db = get_db();
    const existing = db
        .prepare('SELECT c_custkey FROM customer WHERE c_custkey = ?')
        .get(c_custkey);
    if (existing == null) {
        reply.code(404);
        return { error: 'Customer not found' };
    }
    db.prepare('DELETE FROM orders WHERE o_custkey = ?').run(c_custkey);
    db.prepare('DELETE FROM customer WHERE c_custkey = ?').run(c_custkey);
    reply.code(204);
    return;
});

app.get('/api/orders', async (request, reply) => {
    /**
     * Return orders tied to active customers
     */
    const db = get_db();
    const rows = db
        .prepare(
            'SELECT o.o_orderkey, o.o_custkey, o.o_totalprice, o.o_orderdate, o.o_comment ' +
            'FROM orders o JOIN customer c ON o.o_custkey = c.c_custkey ' +
            'ORDER BY o.o_orderdate'
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