const os = require("os");
const express = require("express");
const Database = require("better-sqlite3");

const app = express();
app.use(express.json());

const DATABASE = process.env.DB_PATH || "app.db";
const db = new Database(DATABASE);

function get_db() {
    return db;
}

function close_db(exception) {
    const database = get_db();
    if (database) {
        database.close();
    }
}

process.on("exit", close_db);
process.on("SIGINT", () => {
    close_db();
    process.exit(0);
});
process.on("SIGTERM", () => {
    close_db();
    process.exit(0);
});

app.get("/api/orders/:o_orderkey", (req, res) => {
    const o_orderkey = parseInt(req.params.o_orderkey, 10);
    const db = get_db();
    const row = db.prepare(
        "SELECT o_orderkey, o_orderstatus, o_totalprice, o_orderdate, o_comment " +
        "FROM orders WHERE o_orderkey = ?"
    ).get(o_orderkey);
    if (row === undefined) {
        return res.status(404).json({ error: "Order not found" });
    }
    return res.status(200).json(row);
});

app.post("/api/orders/:o_orderkey/lineitems", (req, res) => {
    const o_orderkey = parseInt(req.params.o_orderkey, 10);
    const payload = req.body || {};
    const db = get_db();

    db.prepare("BEGIN").run();
    try {
        const order = db.prepare(
            "SELECT o_orderkey FROM orders WHERE o_orderkey = ?"
        ).get(o_orderkey);
        if (order === undefined) {
            db.prepare("ROLLBACK").run();
            return res.status(404).json({ error: "Order not found" });
        }
        db.prepare(
            "INSERT INTO lineitem (l_orderkey, l_linenumber, l_extendedprice, l_discount, l_tax, l_shipdate, l_comment) " +
            "VALUES (?, ?, ?, ?, ?, ?, ?)"
        ).run(
            o_orderkey,
            payload["l_linenumber"],
            payload["l_extendedprice"],
            payload["l_discount"],
            payload["l_tax"],
            payload["l_shipdate"] !== undefined ? payload["l_shipdate"] : null,
            payload["l_comment"] !== undefined ? payload["l_comment"] : ""
        );
        db.prepare("COMMIT").run();
        return res.status(201).json({ l_orderkey: o_orderkey, l_linenumber: payload["l_linenumber"] });
    } catch (err) {
        try {
            db.prepare("ROLLBACK").run();
        } catch (_) {}
        throw err;
    }
});

app.post("/api/orders", (req, res) => {
    const payload = req.body || {};
    const db = get_db();

    db.prepare("BEGIN").run();
    try {
        const cur = db.prepare(
            "INSERT INTO orders (o_orderstatus, o_totalprice, o_orderdate, o_comment) " +
            "VALUES (?, 0, ?, ?)"
        ).run(
            payload["o_orderstatus"] !== undefined ? payload["o_orderstatus"] : "O",
            payload["o_orderdate"],
            payload["o_comment"] !== undefined ? payload["o_comment"] : ""
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
                item["l_shipdate"] !== undefined ? item["l_shipdate"] : null,
                item["l_comment"] !== undefined ? item["l_comment"] : ""
            );
        }
        const totalRow = db.prepare(
            "SELECT COALESCE(SUM(l_extendedprice * (1 - l_discount) * (1 + l_tax)), 0) AS total " +
            "FROM lineitem WHERE l_orderkey = ?"
        ).get(o_orderkey);
        const total = totalRow["total"];
        db.prepare(
            "UPDATE orders SET o_totalprice = ? WHERE o_orderkey = ?"
        ).run(total, o_orderkey);
        db.prepare("COMMIT").run();
        return res.status(201).json({ o_orderkey: o_orderkey, o_totalprice: total });
    } catch (err) {
        try {
            db.prepare("ROLLBACK").run();
        } catch (_) {}
        throw err;
    }
});

if (require.main === module) {
    app.listen(parseInt(process.env.PORT || "5000", 10), "0.0.0.0");
}