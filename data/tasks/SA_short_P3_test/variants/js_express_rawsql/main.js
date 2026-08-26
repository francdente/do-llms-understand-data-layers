const os = require("os");
const express = require("express");
const Database = require("better-sqlite3");

const app = express();
const DATABASE = process.env.DB_PATH || "app.db";

function get_db() {
    return new Database(DATABASE);
}

function close_db(db) {
    if (db) {
        db.close();
    }
}

app.post("/api/orders/:o_orderkey/lineitems/:l_linenumber/ship", (req, res) => {
    const o_orderkey = parseInt(req.params.o_orderkey, 10);
    const l_linenumber = parseInt(req.params.l_linenumber, 10);
    const db = get_db();
    try {
        const existing = db.prepare(
            "SELECT l_orderkey FROM lineitem WHERE l_orderkey = ? AND l_linenumber = ?"
        ).get(o_orderkey, l_linenumber);
        if (existing === undefined) {
            return res.status(404).json({ error: "Line item not found" });
        }
        const today = new Date().toISOString().slice(0, 10);
        db.prepare(
            "UPDATE lineitem SET l_shipdate = ? WHERE l_orderkey = ? AND l_linenumber = ?"
        ).run(today, o_orderkey, l_linenumber);
        return res.status(200).json({
            l_orderkey: o_orderkey,
            l_linenumber: l_linenumber,
            l_shipdate: today
        });
    } finally {
        close_db(db);
    }
});

app.get("/api/orders/:o_orderkey", (req, res) => {
    const o_orderkey = parseInt(req.params.o_orderkey, 10);
    const db = get_db();
    try {
        const row = db.prepare(
            "SELECT o_orderkey, o_totalprice, o_orderdate, o_latest_shipdate, o_comment FROM orders WHERE o_orderkey = ?"
        ).get(o_orderkey);
        if (row === undefined) {
            return res.status(404).json({ error: "Order not found" });
        }
        return res.status(200).json(row);
    } finally {
        close_db(db);
    }
});

if (require.main === module) {
    app.listen(parseInt(process.env.PORT || "5000", 10), "0.0.0.0");
}