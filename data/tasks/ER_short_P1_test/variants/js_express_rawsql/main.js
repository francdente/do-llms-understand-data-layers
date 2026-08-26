const os = require("os");
const express = require("express");
const Database = require("better-sqlite3");

const app = express();
const DATABASE = process.env.DB_PATH || "app.db";

app.use(express.json());

const db = new Database(DATABASE);

function close_db(exception) {
    if (db) {
        db.close();
    }
}

process.on("SIGINT", () => {
    close_db();
    process.exit(0);
});

process.on("SIGTERM", () => {
    close_db();
    process.exit(0);
});

app.post("/api/products/:p_productkey/discontinue", (req, res) => {
    const p_productkey = parseInt(req.params.p_productkey, 10);
    const existing = db
        .prepare("SELECT p_productkey FROM product WHERE p_productkey = ?")
        .get(p_productkey);
    if (existing === undefined) {
        return res.status(404).json({ error: "Product not found" });
    }
    db.prepare(
        "UPDATE product SET p_status = 'discontinued' WHERE p_productkey = ?"
    ).run(p_productkey);
    return res.status(200).json({ p_productkey: p_productkey, p_status: "discontinued" });
});

app.get("/api/products/:p_productkey", (req, res) => {
    /**
     * Return purchasable product
     */
    const p_productkey = parseInt(req.params.p_productkey, 10);
    const row = db
        .prepare(
            "SELECT p_productkey, p_name, p_status, p_retailprice " +
            "FROM product WHERE p_productkey = ?"
        )
        .get(p_productkey);
    if (row === undefined) {
        return res.status(404).json({ error: "Product not found" });
    }
    return res.status(200).json(row);
});

const port = parseInt(process.env.PORT || "5000", 10);
app.listen(port, "0.0.0.0");