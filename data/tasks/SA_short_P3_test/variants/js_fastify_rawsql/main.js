const path = require("path");
const Fastify = require("fastify");
const Database = require("better-sqlite3");

const app = Fastify({ logger: false });
const DATABASE = process.env.DB_PATH || "app.db";

function get_db() {
    return new Database(DATABASE);
}

app.post("/api/orders/:o_orderkey/lineitems/:l_linenumber/ship", async (request, reply) => {
    const o_orderkey = Number(request.params.o_orderkey);
    const l_linenumber = Number(request.params.l_linenumber);
    const db = get_db();
    try {
        const existing = db.prepare(
            "SELECT l_orderkey FROM lineitem WHERE l_orderkey = ? AND l_linenumber = ?"
        ).get(o_orderkey, l_linenumber);
        if (existing === undefined) {
            reply.code(404);
            return { error: "Line item not found" };
        }
        const today = new Date().toISOString().slice(0, 10);
        db.prepare(
            "UPDATE lineitem SET l_shipdate = ? WHERE l_orderkey = ? AND l_linenumber = ?"
        ).run(today, o_orderkey, l_linenumber);
        reply.code(200);
        return { l_orderkey: o_orderkey, l_linenumber: l_linenumber, l_shipdate: today };
    } finally {
        db.close();
    }
});

app.get("/api/orders/:o_orderkey", async (request, reply) => {
    const o_orderkey = Number(request.params.o_orderkey);
    const db = get_db();
    try {
        const row = db.prepare(
            "SELECT o_orderkey, o_totalprice, o_orderdate, o_latest_shipdate, o_comment FROM orders WHERE o_orderkey = ?"
        ).get(o_orderkey);
        if (row === undefined) {
            reply.code(404);
            return { error: "Order not found" };
        }
        reply.code(200);
        return row;
    } finally {
        db.close();
    }
});

const port = Number(process.env.PORT || 5000);
app.listen({ host: "0.0.0.0", port }).catch((err) => {
    console.error(err);
    process.exit(1);
});