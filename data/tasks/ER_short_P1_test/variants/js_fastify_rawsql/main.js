const path = require("path");
const Fastify = require("fastify");
const Database = require("better-sqlite3");

const app = Fastify();
const DATABASE = process.env.DB_PATH || "app.db";

const db = new Database(DATABASE);

app.post("/api/products/:p_productkey/discontinue", async (request, reply) => {
  const p_productkey = Number(request.params.p_productkey);

  const existing = db
    .prepare("SELECT p_productkey FROM product WHERE p_productkey = ?")
    .get(p_productkey);

  if (existing == null) {
    reply.code(404);
    return { error: "Product not found" };
  }

  db.prepare(
    "UPDATE product SET p_status = 'discontinued' WHERE p_productkey = ?"
  ).run(p_productkey);

  reply.code(200);
  return { p_productkey, p_status: "discontinued" };
});

app.get("/api/products/:p_productkey", async (request, reply) => {
  /**
   * Return purchasable product
   */
  const p_productkey = Number(request.params.p_productkey);

  const row = db
    .prepare(
      "SELECT p_productkey, p_name, p_status, p_retailprice " +
        "FROM product WHERE p_productkey = ?"
    )
    .get(p_productkey);

  if (row == null) {
    reply.code(404);
    return { error: "Product not found" };
  }

  reply.code(200);
  return row;
});

const port = Number(process.env.PORT || 5000);
app.listen({ host: "0.0.0.0", port }).catch((err) => {
  console.error(err);
  process.exit(1);
});