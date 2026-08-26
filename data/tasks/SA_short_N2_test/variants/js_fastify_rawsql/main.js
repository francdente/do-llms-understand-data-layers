const os = require("os");
const Fastify = require("fastify");
const Database = require("better-sqlite3");

const app = Fastify();
const DATABASE = process.env.DB_PATH || "app.db";

function get_db() {
    return new Database(DATABASE);
}

function close_db(db) {
    if (db) {
        db.close();
    }
}

app.get("/api/orders/:o_orderkey", async (request, reply) => {
    const o_orderkey = Number(request.params.o_orderkey);
    const db = get_db();
    try {
        db.exec("BEGIN");
        const row = db.prepare(
            "SELECT o_orderkey, o_orderstatus, o_orderdate, o_comment " +
            "FROM orders WHERE o_orderkey = ?"
        ).get(o_orderkey);
        if (row == null) {
            db.exec("ROLLBACK");
            return reply.code(404).send({ error: "Order not found" });
        }
        const total = db.prepare(
            "SELECT COALESCE(SUM(l_extendedprice * (1 - l_discount) * (1 + l_tax)), 0) AS total " +
            "FROM lineitem WHERE l_orderkey = ?"
        ).get(o_orderkey);
        db.exec("COMMIT");
        const result = { ...row };
        result.o_totalprice = total.total;
        return reply.code(200).send(result);
    } catch (err) {
        try {
            db.exec("ROLLBACK");
        } catch (_) {}
        throw err;
    } finally {
        close_db(db);
    }
});

app.put("/api/orders/:o_orderkey/lineitems/:l_linenumber", async (request, reply) => {
    const o_orderkey = Number(request.params.o_orderkey);
    const l_linenumber = Number(request.params.l_linenumber);
    const payload = request.body || {};
    const db = get_db();
    try {
        db.exec("BEGIN");
        const existing = db.prepare(
            "SELECT l_orderkey FROM lineitem WHERE l_orderkey = ? AND l_linenumber = ?"
        ).get(o_orderkey, l_linenumber);
        if (existing == null) {
            db.exec("ROLLBACK");
            return reply.code(404).send({ error: "Line item not found" });
        }
        db.prepare(
            "UPDATE lineitem SET l_extendedprice = ?, l_discount = ?, l_tax = ? " +
            "WHERE l_orderkey = ? AND l_linenumber = ?"
        ).run(
            payload["l_extendedprice"], payload["l_discount"], payload["l_tax"],
            o_orderkey, l_linenumber
        );
        const row = db.prepare(
            "SELECT l_orderkey, l_linenumber, l_extendedprice, l_discount, l_tax " +
            "FROM lineitem WHERE l_orderkey = ? AND l_linenumber = ?"
        ).get(o_orderkey, l_linenumber);
        db.exec("COMMIT");
        return reply.code(200).send(row);
    } catch (err) {
        try {
            db.exec("ROLLBACK");
        } catch (_) {}
        throw err;
    } finally {
        close_db(db);
    }
});

const port = Number(process.env.PORT || 5000);
app.listen({ host: "0.0.0.0", port });