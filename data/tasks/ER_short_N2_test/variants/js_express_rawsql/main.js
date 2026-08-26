const express = require("express");
const Database = require("better-sqlite3");

const app = express();
const DATABASE = process.env.DB_PATH || "app.db";
const PORT = parseInt(process.env.PORT || "5000", 10);

const db = new Database(DATABASE);
db.pragma("journal_mode = WAL");

app.use(express.json());

app.put("/api/products/:p_productkey/price", (req, res) => {
  const p_productkey = parseInt(req.params.p_productkey, 10);
  const payload = req.body || {};

  const existing = db
    .prepare("SELECT p_productkey FROM product WHERE p_productkey = ?")
    .get(p_productkey);

  if (existing === undefined) {
    return res.status(404).json({ error: "Product not found" });
  }

  db.prepare(
    "UPDATE product SET p_retailprice = ? WHERE p_productkey = ?"
  ).run(payload["p_retailprice"], p_productkey);

  const row = db
    .prepare(
      "SELECT p_productkey, p_name, p_status, p_retailprice FROM product WHERE p_productkey = ?"
    )
    .get(p_productkey);

  return res.status(200).json(row);
});

app.get("/api/products/:p_productkey", (req, res) => {
  const p_productkey = parseInt(req.params.p_productkey, 10);

  const row = db
    .prepare(
      "SELECT p_productkey, p_name, p_status, p_retailprice FROM product WHERE p_productkey = ?"
    )
    .get(p_productkey);

  if (row === undefined) {
    return res.status(404).json({ error: "Product not found" });
  }

  return res.status(200).json(row);
});

app.listen(PORT, "0.0.0.0");