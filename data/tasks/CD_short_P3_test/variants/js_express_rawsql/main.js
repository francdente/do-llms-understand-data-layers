const os = require("os");
const express = require("express");
const Database = require("better-sqlite3");

const app = express();
const DATABASE = process.env.DB_PATH || "app.db";

function getDb() {
    if (!app.locals._database) {
        app.locals._database = new Database(DATABASE);
        app.locals._database.pragma("foreign_keys = ON");
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

app.delete("/api/customers/:c_custkey", (req, res) => {
    const c_custkey = parseInt(req.params.c_custkey, 10);
    const db = getDb();
    const existing = db
        .prepare("SELECT c_custkey FROM customer WHERE c_custkey = ?")
        .get(c_custkey);
    if (existing === undefined) {
        return res.status(404).json({ error: "Customer not found" });
    }

    const tx = db.transaction((custkey) => {
        db.prepare("DELETE FROM orders WHERE o_custkey = ?").run(custkey);
        db.prepare("DELETE FROM customer WHERE c_custkey = ?").run(custkey);
    });
    tx(c_custkey);

    return res.status(204).send("");
});

app.get("/api/orders/monthly-trend", (req, res) => {
    /**
     * Return the trend of all existed orders per-month.
     */
    const db = getDb();
    const rows = db
        .prepare(
            "SELECT strftime('%Y-%m', o_orderdate) AS month, " +
                "       COUNT(*) AS order_count, " +
                "       SUM(o_totalprice) AS revenue " +
                "FROM orders " +
                "GROUP BY month " +
                "ORDER BY month"
        )
        .all();
    return res.status(200).json(rows);
});

if (require.main === module) {
    app.listen(parseInt(process.env.PORT || "5000", 10), "0.0.0.0");
}