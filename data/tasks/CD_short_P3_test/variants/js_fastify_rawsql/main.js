const os = require('os');
const Fastify = require('fastify');
const Database = require('better-sqlite3');

const app = Fastify();
const DATABASE = process.env.DB_PATH || 'app.db';

function get_db() {
    return new Database(DATABASE);
}

function close_db(db) {
    if (db) {
        db.close();
    }
}

app.delete('/api/customers/:c_custkey', async (request, reply) => {
    const c_custkey = parseInt(request.params.c_custkey, 10);
    const db = get_db();
    try {
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
    } finally {
        close_db(db);
    }
});

app.get('/api/orders/monthly-trend', async (request, reply) => {
    /**
     * Return the trend of all existed orders per-month.
     */
    const db = get_db();
    try {
        const rows = db
            .prepare(
                "SELECT strftime('%Y-%m', o_orderdate) AS month, " +
                "       COUNT(*) AS order_count, " +
                "       SUM(o_totalprice) AS revenue " +
                "FROM orders " +
                "GROUP BY month " +
                "ORDER BY month"
            )
            .all();
        reply.code(200);
        return rows;
    } finally {
        close_db(db);
    }
});

const port = parseInt(process.env.PORT || '5000', 10);
app.listen({ host: '0.0.0.0', port }).catch((err) => {
    console.error(err);
    process.exit(1);
});