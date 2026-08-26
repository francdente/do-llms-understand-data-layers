const express = require("express");
const Database = require("better-sqlite3");

const app = express();
const DATABASE = process.env.DB_PATH || "app.db";

function getDb() {
    return new Database(DATABASE);
}

app.delete("/api/customers/:c_custkey", (req, res) => {
    const c_custkey = parseInt(req.params.c_custkey, 10);
    const db = getDb();
    try {
        const existing = db
            .prepare("SELECT c_custkey FROM customer WHERE c_custkey = ?")
            .get(c_custkey);
        if (existing === undefined) {
            db.close();
            return res.status(404).json({ error: "Customer not found" });
        }

        const txn = db.transaction(() => {
            db.prepare("DELETE FROM orders WHERE o_custkey = ?").run(c_custkey);
            db.prepare("DELETE FROM customer WHERE c_custkey = ?").run(c_custkey);
        });
        txn();

        db.close();
        return res.status(204).send("");
    } catch (err) {
        db.close();
        throw err;
    }
});

app.get("/api/orders", (req, res) => {
    /**
     * Return orders tied to active customers
     */
    const db = getDb();
    try {
        const rows = db
            .prepare(
                "SELECT o.o_orderkey, o.o_custkey, o.o_totalprice, o.o_orderdate, o.o_comment " +
                "FROM orders o JOIN customer c ON o.o_custkey = c.c_custkey " +
                "ORDER BY o.o_orderdate"
            )
            .all();
        db.close();
        return res.status(200).json(rows);
    } catch (err) {
        db.close();
        throw err;
    }
});

const port = parseInt(process.env.PORT || "5000", 10);
app.listen(port, "0.0.0.0");