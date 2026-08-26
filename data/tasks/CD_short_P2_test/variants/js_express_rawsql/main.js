const express = require("express");
const Database = require("better-sqlite3");

const app = express();
const DATABASE = process.env.DB_PATH || "app.db";

function get_db() {
    return new Database(DATABASE);
}

app.delete("/api/customers/:c_custkey", (req, res) => {
    const c_custkey = parseInt(req.params.c_custkey, 10);
    const db = get_db();
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

app.get("/api/orders/history", (req, res) => {
    /**
     * Return the history of all existed orders
     */
    const db = get_db();
    try {
        const rows = db
            .prepare(
                "SELECT o_orderkey, o_custkey, o_totalprice, o_orderdate, o_comment " +
                "FROM orders ORDER BY o_orderdate"
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