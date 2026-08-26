const os = require('os');
const Fastify = require('fastify');
const Database = require('better-sqlite3');

const app = Fastify();
const DATABASE = process.env.DB_PATH || 'app.db';

let db = null;

function get_db() {
    if (db === null) {
        db = new Database(DATABASE);
        db.pragma('foreign_keys = ON');
    }
    return db;
}

async function close_db() {
    if (db !== null) {
        db.close();
        db = null;
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
    if (existing === undefined) {
        reply.code(404);
        return { error: 'Customer not found' };
    }
    try {
        db.prepare('DELETE FROM customer WHERE c_custkey = ?').run(c_custkey);
    } catch (err) {
        if (err && err.code === 'SQLITE_CONSTRAINT_FOREIGNKEY') {
            reply.code(409);
            return { error: 'Cannot delete customer with existing orders' };
        }
        throw err;
    }
    reply.code(204);
    return '';
});

app.get('/api/dashboard', async (request, reply) => {
    /**
     * Return dashboard based on all existed orders
     */
    const db = get_db();
    const row = db.prepare(
        'SELECT COUNT(*) AS total_orders, ' +
        '       COALESCE(SUM(o_totalprice), 0) AS total_revenue ' +
        'FROM orders'
    ).get();
    reply.code(200);
    return row;
});

const port = parseInt(process.env.PORT || '5000', 10);

app.listen({ host: '0.0.0.0', port }).catch((err) => {
    console.error(err);
    process.exit(1);
});