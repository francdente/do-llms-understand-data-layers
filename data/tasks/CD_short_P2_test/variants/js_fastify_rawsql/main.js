const path = require("path");
const Fastify = require("fastify");
const Database = require("better-sqlite3");

const app = Fastify();
const DATABASE = process.env.DB_PATH || "app.db";
const db = new Database(path.resolve(DATABASE));

app.delete("/api/customers/:c_custkey", async (request, reply) => {
    const c_custkey = Number(request.params.c_custkey);
    const existing = db
        .prepare("SELECT c_custkey FROM customer WHERE c_custkey = ?")
        .get(c_custkey);
    if (existing === undefined) {
        reply.code(404);
        return { error: "Customer not found" };
    }
    const txn = db.transaction(() => {
        db.prepare("DELETE FROM orders WHERE o_custkey = ?").run(c_custkey);
        db.prepare("DELETE FROM customer WHERE c_custkey = ?").run(c_custkey);
    });
    txn();
    reply.code(204);
    return;
});

app.get("/api/orders/history", async (request, reply) => {
    /**
     * Return the history of all existed orders
     */
    const rows = db
        .prepare(
            "SELECT o_orderkey, o_custkey, o_totalprice, o_orderdate, o_comment " +
            "FROM orders ORDER BY o_orderdate"
        )
        .all();
    reply.code(200);
    return rows;
});

const port = Number(process.env.PORT || 5000);
app.listen({ host: "0.0.0.0", port }).catch((err) => {
    console.error(err);
    process.exit(1);
});