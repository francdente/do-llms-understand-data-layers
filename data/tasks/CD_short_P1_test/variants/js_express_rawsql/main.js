const express = require("express");
const Database = require("better-sqlite3");

const app = express();
const DATABASE = process.env.DB_PATH || "app.db";

const db = new Database(DATABASE);

app.delete("/api/customers/:c_custkey", (req, res) => {
    const c_custkey = parseInt(req.params.c_custkey, 10);
    const existing = db
        .prepare("SELECT c_custkey FROM customer WHERE c_custkey = ?")
        .get(c_custkey);
    if (existing === undefined) {
        return res.status(404).json({ error: "Customer not found" });
    }
    const txn = db.transaction((custkey) => {
        db.prepare("DELETE FROM orders WHERE o_custkey = ?").run(custkey);
        db.prepare("DELETE FROM customer WHERE c_custkey = ?").run(custkey);
    });
    txn(c_custkey);
    return res.status(204).send("");
});

app.get("/api/orders/summary", (req, res) => {
    /**
     * Return the summary of all existed historical orders
     */
    const row = db
        .prepare(
            "SELECT COUNT(*) AS total_orders, " +
            "       COALESCE(SUM(o_totalprice), 0) AS total_revenue " +
            "FROM orders"
        )
        .get();
    return res.status(200).json(row);
});

const port = parseInt(process.env.PORT || "5000", 10);
app.listen(port, "0.0.0.0");