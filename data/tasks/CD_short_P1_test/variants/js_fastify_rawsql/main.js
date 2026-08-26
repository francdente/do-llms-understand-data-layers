const os = require("os");
const Fastify = require("fastify");
const Database = require("better-sqlite3");

const app = Fastify();
const DATABASE = process.env.DB_PATH || "app.db";
const db = new Database(DATABASE);

app.delete("/api/customers/:c_custkey", async (request, reply) => {
    const c_custkey = Number(request.params.c_custkey);
    const existing = db
        .prepare("SELECT c_custkey FROM customer WHERE c_custkey = ?")
        .get(c_custkey);
    if (existing === undefined) {
        reply.code(404).send({ error: "Customer not found" });
        return;
    }
    const txn = db.transaction((custkey) => {
        db.prepare("DELETE FROM orders WHERE o_custkey = ?").run(custkey);
        db.prepare("DELETE FROM customer WHERE c_custkey = ?").run(custkey);
    });
    txn(c_custkey);
    reply.code(204).send();
});

app.get("/api/orders/summary", async (request, reply) => {
    /**
     * Return the summary of all existed historical orders
     */
    const row = db
        .prepare(
            "SELECT COUNT(*) AS total_orders, " +
            "       COALESCE(SUM(o_totalprice), 0) AS total_revenue " +
            "FROM orders"
        )
        .get();
    reply.code(200).send(row);
});

const port = Number(process.env.PORT || 5000);
app.listen({ host: "0.0.0.0", port }, (err) => {
    if (err) {
        process.exit(1);
    }
});