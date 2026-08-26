const express = require("express");
const Database = require("better-sqlite3");

const app = express();
const DATABASE = process.env.DB_PATH || "app.db";
const PORT = parseInt(process.env.PORT || "5000", 10);

app.use(express.json());

const db = new Database(DATABASE);

app.post("/api/products/:p_productkey/archive", (req, res) => {
  const p_productkey = parseInt(req.params.p_productkey, 10);

  const existing = db
    .prepare("SELECT p_productkey FROM product WHERE p_productkey = ?")
    .get(p_productkey);

  if (existing === undefined) {
    return res.status(404).json({ error: "Product not found" });
  }

  db.prepare(
    "UPDATE product SET p_status = 'archived' WHERE p_productkey = ?"
  ).run(p_productkey);

  return res.status(200).json({ p_productkey, p_status: "archived" });
});

app.get("/api/products/archived", (req, res) => {
  const rows = db
    .prepare(
      "SELECT p_productkey, p_name, p_retailprice " +
        "FROM product WHERE p_status = 'archived' " +
        "ORDER BY p_productkey"
    )
    .all();

  return res.status(200).json(rows);
});

process.on("SIGINT", () => {
  try {
    db.close();
  } finally {
    process.exit(0);
  }
});

process.on("SIGTERM", () => {
  try {
    db.close();
  } finally {
    process.exit(0);
  }
});

app.listen(PORT, "0.0.0.0");