const express = require("express");
const Database = require("better-sqlite3");

const app = express();
const DATABASE = process.env.DB_PATH || "app.db";

app.use(express.json());

function getDb() {
    return new Database(DATABASE);
}

app.post("/api/products/:p_productkey/discontinue", (req, res) => {
    const p_productkey = parseInt(req.params.p_productkey, 10);
    const db = getDb();
    try {
        const existing = db
            .prepare("SELECT p_productkey FROM product WHERE p_productkey = ?")
            .get(p_productkey);
        if (existing === undefined) {
            db.close();
            return res.status(404).json({ error: "Product not found" });
        }
        db.prepare(
            "UPDATE product SET p_status = 'discontinued' WHERE p_productkey = ?"
        ).run(p_productkey);
        db.close();
        return res.status(200).json({ p_productkey: p_productkey, p_status: "discontinued" });
    } catch (err) {
        db.close();
        throw err;
    }
});

app.get("/api/products", (req, res) => {
    /**
     * List purchasable products
     */
    const db = getDb();
    try {
        const rows = db
            .prepare(
                "SELECT p_productkey, p_name, p_retailprice " +
                "FROM product WHERE p_status = 'active' " +
                "ORDER BY p_productkey"
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