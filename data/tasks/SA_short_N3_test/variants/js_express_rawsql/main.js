const os = require("os");
const express = require("express");
const Database = require("better-sqlite3");

const app = express();
const DATABASE = process.env.DB_PATH || "app.db";

function getDb() {
    if (!app.locals._database) {
        app.locals._database = new Database(DATABASE);
    }
    return app.locals._database;
}

function closeDb() {
    const db = app.locals._database;
    if (db) {
        db.close();
        app.locals._database = null;
    }
}

process.on("exit", closeDb);
process.on("SIGINT", () => {
    closeDb();
    process.exit(0);
});
process.on("SIGTERM", () => {
    closeDb();
    process.exit(0);
});

app.post("/api/orders/:o_orderkey/lineitems/:l_linenumber/ship", (req, res) => {
    const o_orderkey = parseInt(req.params.o_orderkey, 10);
    const l_linenumber = parseInt(req.params.l_linenumber, 10);
    const db = getDb();

    const existing = db.prepare(
        "SELECT l_orderkey FROM lineitem WHERE l_orderkey = ? AND l_linenumber = ?"
    ).get(o_orderkey, l_linenumber);

    if (existing === undefined) {
        return res.status(404).json({ error: "Line item not found" });
    }

    const today = new Date().toISOString().slice(0, 10);

    const tx = db.transaction(() => {
        db.prepare(
            "UPDATE lineitem SET l_shipdate = ? WHERE l_orderkey = ? AND l_linenumber = ?"
        ).run(today, o_orderkey, l_linenumber);

        const latestRow = db.prepare(
            "SELECT MAX(l_shipdate) AS latest FROM lineitem WHERE l_orderkey = ?"
        ).get(o_orderkey);

        const latest = latestRow.latest;

        db.prepare(
            "UPDATE orders SET o_latest_shipdate = ? WHERE o_orderkey = ?"
        ).run(latest, o_orderkey);
    });

    tx();

    return res.status(200).json({
        l_orderkey: o_orderkey,
        l_linenumber: l_linenumber,
        l_shipdate: today
    });
});

app.get("/api/orders/:o_orderkey", (req, res) => {
    const o_orderkey = parseInt(req.params.o_orderkey, 10);
    const db = getDb();

    const row = db.prepare(
        "SELECT o_orderkey, o_totalprice, o_orderdate, o_latest_shipdate, o_comment FROM orders WHERE o_orderkey = ?"
    ).get(o_orderkey);

    if (row === undefined) {
        return res.status(404).json({ error: "Order not found" });
    }

    return res.status(200).json(row);
});

if (require.main === module) {
    app.listen(parseInt(process.env.PORT || "5000", 10), "0.0.0.0");
}