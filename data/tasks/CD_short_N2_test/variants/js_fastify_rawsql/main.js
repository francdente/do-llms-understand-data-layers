const path = require("path");
const Fastify = require("fastify");
const Database = require("better-sqlite3");

const app = Fastify();
const DATABASE = process.env.DB_PATH || "app.db";

function get_db() {
    return new Database(DATABASE);
}

app.delete("/api/customers/:c_custkey", async (request, reply) => {
    const c_custkey = Number(request.params.c_custkey);
    const db = get_db();
    try {
        const existing = db
            .prepare("SELECT c_custkey FROM customer WHERE c_custkey = ?")
            .get(c_custkey);
        if (existing == null) {
            reply.code(404).send({ error: "Customer not found" });
            return;
        }
        const tx = db.transaction((custkey) => {
            db.prepare("DELETE FROM orders WHERE o_custkey = ?").run(custkey);
            db.prepare("DELETE FROM customer WHERE c_custkey = ?").run(custkey);
        });
        tx(c_custkey);
        reply.code(204).send();
    } finally {
        db.close();
    }
});

app.get("/api/orders", async (request, reply) => {
    /**
     * Return orders tied to active customers
     */
    const db = get_db();
    try {
        const rows = db
            .prepare(
                "SELECT o_orderkey, o_custkey, o_totalprice, o_orderdate, o_comment " +
                "FROM orders WHERE o_custkey IN (SELECT c_custkey FROM customer) " +
                "ORDER BY o_orderdate"
            )
            .all();
        reply.code(200).send(rows);
    } finally {
        db.close();
    }
});

const port = Number(process.env.PORT || 5000);
app.listen({ host: "0.0.0.0", port });