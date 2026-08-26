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

app.post("/api/products/:p_productkey/discontinue", (req, res) => {
    const p_productkey = parseInt(req.params.p_productkey, 10);
    const db = get_db();
    try {
        const existing = db
            .prepare("SELECT p_productkey FROM product WHERE p_productkey = ?")
            .get(p_productkey);
        if (existing == null) {
            return res.status(404).json({ error: "Product not found" });
        }
        db.prepare(
            "UPDATE product SET p_status = 'discontinued' WHERE p_productkey = ?"
        ).run(p_productkey);
        return res.status(200).json({ p_productkey: p_productkey, p_status: "discontinued" });
    } finally {
        close_db(db);
    }
});

app.get("/api/products/:p_productkey/availability", (req, res) => {
    /**
     * Get availability for purchasable product
     */
    const p_productkey = parseInt(req.params.p_productkey, 10);
    const db = get_db();
    try {
        const row = db
            .prepare(
                "SELECT p.p_productkey, p.p_name, " +
                "       COALESCE(SUM(i.i_quantity), 0) AS total_stock " +
                "FROM product p LEFT JOIN inventory i ON p.p_productkey = i.i_productkey " +
                "WHERE p.p_productkey = ? " +
                "GROUP BY p.p_productkey"
            )
            .get(p_productkey);
        if (row == null) {
            return res.status(404).json({ error: "Product not found" });
        }
        return res.status(200).json(row);
    } finally {
        close_db(db);
    }
});

if (require.main === module) {
    app.listen(parseInt(process.env.PORT || "5000", 10), "0.0.0.0");
}