const os = require("os");
const express = require("express");
const Database = require("better-sqlite3");

const app = express();
app.use(express.json());

const DATABASE = process.env.DB_PATH || "app.db";
const db = new Database(DATABASE);

function runInTransaction(fn) {
    db.exec("BEGIN");
    try {
        const result = fn();
        db.exec("COMMIT");
        return result;
    } catch (err) {
        try {
            db.exec("ROLLBACK");
        } catch (_) {}
        throw err;
    }
}

app.get("/api/orders/:o_orderkey", (req, res) => {
    const o_orderkey = parseInt(req.params.o_orderkey, 10);

    try {
        db.exec("BEGIN");
        const row = db.prepare(
            "SELECT o_orderkey, o_orderstatus, o_orderdate, o_comment " +
            "FROM orders WHERE o_orderkey = ?"
        ).get(o_orderkey);

        if (row == null) {
            db.exec("ROLLBACK");
            return res.status(404).json({ error: "Order not found" });
        }

        const total = db.prepare(
            "SELECT COALESCE(SUM(l_extendedprice * (1 - l_discount) * (1 + l_tax)), 0) AS total " +
            "FROM lineitem WHERE l_orderkey = ?"
        ).get(o_orderkey);

        db.exec("COMMIT");
        const result = { ...row, o_totalprice: total.total };
        return res.status(200).json(result);
    } catch (err) {
        try {
            db.exec("ROLLBACK");
        } catch (_) {}
        return res.status(500).json({ error: "Internal Server Error" });
    }
});

app.put("/api/orders/:o_orderkey/lineitems/:l_linenumber", (req, res) => {
    const o_orderkey = parseInt(req.params.o_orderkey, 10);
    const l_linenumber = parseInt(req.params.l_linenumber, 10);
    const payload = req.body && typeof req.body === "object" ? req.body : {};

    try {
        db.exec("BEGIN");
        const existing = db.prepare(
            "SELECT l_orderkey FROM lineitem WHERE l_orderkey = ? AND l_linenumber = ?"
        ).get(o_orderkey, l_linenumber);

        if (existing == null) {
            db.exec("ROLLBACK");
            return res.status(404).json({ error: "Line item not found" });
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

        db.exec("COMMIT");
        return res.status(200).json(row);
    } catch (err) {
        try {
            db.exec("ROLLBACK");
        } catch (_) {}
        return res.status(500).json({ error: "Internal Server Error" });
    }
});

const port = parseInt(process.env.PORT || "5000", 10);
app.listen(port, "0.0.0.0");