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

app.post('/api/products/:p_productkey/discontinue', async (request, reply) => {
    const p_productkey = parseInt(request.params.p_productkey, 10);
    const db = get_db();
    const existing = db
        .prepare('SELECT p_productkey FROM product WHERE p_productkey = ?')
        .get(p_productkey);
    if (existing === undefined) {
        reply.code(404);
        return { error: 'Product not found' };
    }
    db.prepare(
        "UPDATE product SET p_status = 'discontinued' WHERE p_productkey = ?"
    ).run(p_productkey);
    reply.code(200);
    return { p_productkey: p_productkey, p_status: 'discontinued' };
});

app.get('/api/products/:p_productkey/availability', async (request, reply) => {
    /**
     * Get availability for purchasable product
     */
    const p_productkey = parseInt(request.params.p_productkey, 10);
    const db = get_db();
    const row = db.prepare(
        "SELECT p.p_productkey, p.p_name, " +
        "       COALESCE(SUM(i.i_quantity), 0) AS total_stock " +
        "FROM product p LEFT JOIN inventory i ON p.p_productkey = i.i_productkey " +
        "WHERE p.p_productkey = ? " +
        "GROUP BY p.p_productkey"
    ).get(p_productkey);
    if (row === undefined) {
        reply.code(404);
        return { error: 'Product not found' };
    }
    reply.code(200);
    return row;
});

const start = async () => {
    try {
        await app.listen({
            host: '0.0.0.0',
            port: parseInt(process.env.PORT || '5000', 10),
        });
    } catch (err) {
        console.error(err);
        process.exit(1);
    }
};

start();