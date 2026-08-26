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

app.post("/api/suppliers/:s_suppkey/suspend", (req, res) => {
    const s_suppkey = parseInt(req.params.s_suppkey, 10);
    const db = get_db();
    try {
        const existing = db
            .prepare("SELECT s_suppkey FROM supplier WHERE s_suppkey = ?")
            .get(s_suppkey);
        if (existing == null) {
            return res.status(404).json({ error: "Supplier not found" });
        }
        db.prepare(
            "UPDATE supplier SET s_status = 'suspended' WHERE s_suppkey = ?"
        ).run(s_suppkey);
        return res.status(200).json({ s_suppkey: s_suppkey, s_status: "suspended" });
    } finally {
        close_db(db);
    }
});

app.get("/api/products", (req, res) => {
    /**
     * List purchasable products
     */
    const db = get_db();
    try {
        const rows = db
            .prepare(
                "SELECT p.p_productkey, p.p_name, p.p_retailprice, s.s_name AS supplier_name " +
                "FROM product p JOIN supplier s ON p.p_suppkey = s.s_suppkey " +
                "ORDER BY p.p_productkey"
            )
            .all();
        return res.status(200).json(rows);
    } finally {
        close_db(db);
    }
});

const port = parseInt(process.env.PORT || "5000", 10);
app.listen(port, "0.0.0.0");