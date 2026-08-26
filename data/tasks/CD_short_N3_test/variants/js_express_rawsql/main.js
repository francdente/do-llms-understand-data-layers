const express = require("express");
const Database = require("better-sqlite3");

const app = express();
const DATABASE = process.env.DB_PATH || "app.db";

const db = new Database(DATABASE);
db.pragma("foreign_keys = ON");

app.delete("/api/customers/:c_custkey", (req, res) => {
    const c_custkey = parseInt(req.params.c_custkey, 10);
    const existing = db
        .prepare("SELECT c_custkey FROM customer WHERE c_custkey = ?")
        .get(c_custkey);
    if (existing === undefined) {
        return res.status(404).json({ error: "Customer not found" });
    }
    try {
        db.prepare("DELETE FROM customer WHERE c_custkey = ?").run(c_custkey);
    } catch (err) {
        if (err && err.code === "SQLITE_CONSTRAINT_FOREIGNKEY") {
            return res.status(409).json({ error: "Cannot delete customer with existing orders" });
        }
        throw err;
    }
    return res.status(204).send("");
});

app.get("/api/dashboard", (req, res) => {
    /**
     * Return dashboard based on all existed orders
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