const os = require("os");
const Fastify = require("fastify");
const Database = require("better-sqlite3");

const app = Fastify();
const DATABASE = process.env.DB_PATH || "app.db";
const db = new Database(DATABASE);

app.get("/api/orders/:o_orderkey", async (request, reply) => {
    const o_orderkey = Number(request.params.o_orderkey);
    const row = db.prepare(
        "SELECT o_orderkey, o_orderstatus, o_totalprice, o_orderdate, o_comment " +
        "FROM orders WHERE o_orderkey = ?"
    ).get(o_orderkey);
    if (row === undefined) {
        reply.code(404);
        return { error: "Order not found" };
    }
    reply.code(200);
    return row;
});

app.put("/api/orders/:o_orderkey/lineitems/:l_linenumber", async (request, reply) => {
    const o_orderkey = Number(request.params.o_orderkey);
    const l_linenumber = Number(request.params.l_linenumber);
    const payload = request.body || {};

    const txn = db.transaction(() => {
        const existing = db.prepare(
            "SELECT l_orderkey FROM lineitem WHERE l_orderkey = ? AND l_linenumber = ?"
        ).get(o_orderkey, l_linenumber);

        if (existing === undefined) {
            return { notFound: true };
        }

        db.prepare(
            "UPDATE lineitem SET l_extendedprice = ?, l_discount = ?, l_tax = ? " +
            "WHERE l_orderkey = ? AND l_linenumber = ?"
        ).run(
            payload["l_extendedprice"],
            payload["l_discount"],
            payload["l_tax"],
            o_orderkey,
            l_linenumber
        );

        const row = db.prepare(
            "SELECT l_orderkey, l_linenumber, l_extendedprice, l_discount, l_tax " +
            "FROM lineitem WHERE l_orderkey = ? AND l_linenumber = ?"
        ).get(o_orderkey, l_linenumber);

        return { row };
    });

    const result = txn();

    if (result.notFound) {
        reply.code(404);
        return { error: "Line item not found" };
    }

    reply.code(200);
    return result.row;
});

app.post("/api/orders", async (request, reply) => {
    const payload = request.body || {};

    const txn = db.transaction(() => {
        const cur = db.prepare(
            "INSERT INTO orders (o_orderstatus, o_totalprice, o_orderdate, o_comment) " +
            "VALUES (?, 0, ?, ?)"
        ).run(
            payload.o_orderstatus !== undefined ? payload.o_orderstatus : "O",
            payload["o_orderdate"],
            payload.o_comment !== undefined ? payload.o_comment : ""
        );

        const o_orderkey = Number(cur.lastInsertRowid);

        for (const item of payload["lineitems"]) {
            db.prepare(
                "INSERT INTO lineitem (l_orderkey, l_linenumber, l_extendedprice, " +
                "l_discount, l_tax, l_shipdate, l_comment) " +
                "VALUES (?, ?, ?, ?, ?, ?, ?)"
            ).run(
                o_orderkey,
                item["l_linenumber"],
                item["l_extendedprice"],
                item["l_discount"],
                item["l_tax"],
                item["l_shipdate"],
                item.l_comment !== undefined ? item.l_comment : ""
            );
        }

        const totalRow = db.prepare(
            "SELECT COALESCE(SUM(l_extendedprice * (1 - l_discount) * (1 + l_tax)), 0) AS total " +
            "FROM lineitem WHERE l_orderkey = ?"
        ).get(o_orderkey);

        const total = totalRow.total;

        db.prepare(
            "UPDATE orders SET o_totalprice = ? WHERE o_orderkey = ?"
        ).run(total, o_orderkey);

        return { o_orderkey, o_totalprice: total };
    });

    const result = txn();
    reply.code(201);
    return result;
});

const port = Number(process.env.PORT || 5000);
app.listen({ host: "0.0.0.0", port }).catch((err) => {
    console.error(err);
    process.exit(1);
});