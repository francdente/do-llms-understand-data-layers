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

app.post("/api/orders/:o_orderkey/lineitems/:l_linenumber/ship", async (request, reply) => {
    const o_orderkey = parseInt(request.params.o_orderkey, 10);
    const l_linenumber = parseInt(request.params.l_linenumber, 10);
    const db = get_db();
    try {
        const existing = db.prepare(
            "SELECT l_orderkey FROM lineitem WHERE l_orderkey = ? AND l_linenumber = ?"
        ).get(o_orderkey, l_linenumber);
        if (existing == null) {
            return reply.code(404).send({ error: "Line item not found" });
        }
        const today = new Date().toISOString().slice(0, 10);
        db.prepare(
            "UPDATE lineitem SET l_shipdate = ? WHERE l_orderkey = ? AND l_linenumber = ?"
        ).run(today, o_orderkey, l_linenumber);
        const latestRow = db.prepare(
            "SELECT MAX(l_shipdate) AS latest FROM lineitem WHERE l_orderkey = ?"
        ).get(o_orderkey);
        const latest = latestRow ? latestRow.latest : null;
        db.prepare(
            "UPDATE orders SET o_latest_shipdate = ? WHERE o_orderkey = ?"
        ).run(latest, o_orderkey);
        return reply.code(200).send({
            l_orderkey: o_orderkey,
            l_linenumber: l_linenumber,
            l_shipdate: today
        });
    } finally {
        close_db(db);
    }
});

app.get("/api/orders/:o_orderkey", async (request, reply) => {
    const o_orderkey = parseInt(request.params.o_orderkey, 10);
    const db = get_db();
    try {
        const row = db.prepare(
            "SELECT o_orderkey, o_totalprice, o_orderdate, o_latest_shipdate, o_comment FROM orders WHERE o_orderkey = ?"
        ).get(o_orderkey);
        if (row == null) {
            return reply.code(404).send({ error: "Order not found" });
        }
        return reply.code(200).send(row);
    } finally {
        close_db(db);
    }
});

const port = parseInt(process.env.PORT || "5000", 10);
app.listen({ host: "0.0.0.0", port }).catch((err) => {
    console.error(err);
    process.exit(1);
});