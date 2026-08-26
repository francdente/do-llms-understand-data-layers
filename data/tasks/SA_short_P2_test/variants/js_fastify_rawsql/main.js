const path = require("path");
const Fastify = require("fastify");
const Database = require("better-sqlite3");

const app = Fastify();
const DATABASE = process.env.DB_PATH || "app.db";

const db = new Database(DATABASE);

app.get("/api/orders/:o_orderkey", async (request, reply) => {
  const o_orderkey = Number(request.params.o_orderkey);
  const row = db
    .prepare(
      "SELECT o_orderkey, o_orderstatus, o_totalprice, o_orderdate, o_comment " +
        "FROM orders WHERE o_orderkey = ?"
    )
    .get(o_orderkey);

  if (row === undefined) {
    reply.code(404);
    return { error: "Order not found" };
  }

  reply.code(200);
  return row;
});

app.post("/api/orders/:o_orderkey/lineitems", async (request, reply) => {
  const o_orderkey = Number(request.params.o_orderkey);
  const payload = request.body || {};

  try {
    const result = db.transaction(() => {
      db.prepare("BEGIN").run();

      const order = db
        .prepare("SELECT o_orderkey FROM orders WHERE o_orderkey = ?")
        .get(o_orderkey);

      if (order === undefined) {
        db.prepare("ROLLBACK").run();
        return { notFound: true };
      }

      db.prepare(
        "INSERT INTO lineitem (l_orderkey, l_linenumber, l_extendedprice, l_discount, l_tax, l_shipdate, l_comment) " +
          "VALUES (?, ?, ?, ?, ?, ?, ?)"
      ).run(
        o_orderkey,
        payload["l_linenumber"],
        payload["l_extendedprice"],
        payload["l_discount"],
        payload["l_tax"],
        payload["l_shipdate"] ?? null,
        Object.prototype.hasOwnProperty.call(payload, "l_comment") ? payload["l_comment"] : ""
      );

      db.prepare("COMMIT").run();

      return {
        l_orderkey: o_orderkey,
        l_linenumber: payload["l_linenumber"],
      };
    })();

    if (result && result.notFound) {
      reply.code(404);
      return { error: "Order not found" };
    }

    reply.code(201);
    return result;
  } catch (err) {
    reply.code(500);
    throw err;
  }
});

app.post("/api/orders", async (request, reply) => {
  const payload = request.body || {};

  try {
    const result = db.transaction(() => {
      db.prepare("BEGIN").run();

      const cur = db
        .prepare(
          "INSERT INTO orders (o_orderstatus, o_totalprice, o_orderdate, o_comment) " +
            "VALUES (?, 0, ?, ?)"
        )
        .run(
          Object.prototype.hasOwnProperty.call(payload, "o_orderstatus") ? payload["o_orderstatus"] : "O",
          payload["o_orderdate"],
          Object.prototype.hasOwnProperty.call(payload, "o_comment") ? payload["o_comment"] : ""
        );

      const o_orderkey = Number(cur.lastInsertRowid);

      for (const item of payload["lineitems"]) {
        db.prepare(
          "INSERT INTO lineitem (l_orderkey, l_linenumber, l_extendedprice, " +
            "l_discount, l_tax, l_shipdate, l_comment) " +
            "VALUES (?, ?, ?, ?, ?, ?, ?)"
        ).run(
          o_orderkey,
          item["l_linenumber"],
          item["l_extendedprice"],
          item["l_discount"],
          item["l_tax"],
          item["l_shipdate"] ?? null,
          Object.prototype.hasOwnProperty.call(item, "l_comment") ? item["l_comment"] : ""
        );
      }

      const totalRow = db
        .prepare(
          "SELECT COALESCE(SUM(l_extendedprice * (1 - l_discount) * (1 + l_tax)), 0) AS total " +
            "FROM lineitem WHERE l_orderkey = ?"
        )
        .get(o_orderkey);

      const total = totalRow.total;

      db.prepare(
        "UPDATE orders SET o_totalprice = ? WHERE o_orderkey = ?"
      ).run(total, o_orderkey);

      db.prepare("COMMIT").run();

      return { o_orderkey, o_totalprice: total };
    })();

    reply.code(201);
    return result;
  } catch (err) {
    reply.code(500);
    throw err;
  }
});

const port = Number(process.env.PORT || 5000);
app.listen({ host: "0.0.0.0", port }).catch((err) => {
  console.error(err);
  process.exit(1);
});