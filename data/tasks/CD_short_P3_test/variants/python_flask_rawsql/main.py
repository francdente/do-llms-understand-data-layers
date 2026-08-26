import os
import sqlite3
from flask import Flask, jsonify, g

app = Flask(__name__)
DATABASE = os.environ.get("DB_PATH", "app.db")


def get_db():
    db = getattr(g, "_database", None)
    if db is None:
        db = g._database = sqlite3.connect(DATABASE)
        db.row_factory = sqlite3.Row
    return db


@app.teardown_appcontext
def close_db(exception):
    db = getattr(g, "_database", None)
    if db is not None:
        db.close()


@app.route("/api/customers/<int:c_custkey>", methods=["DELETE"])
def delete_customer(c_custkey):
    db = get_db()
    existing = db.execute(
        "SELECT c_custkey FROM customer WHERE c_custkey = ?", (c_custkey,)
    ).fetchone()
    if existing is None:
        return jsonify({"error": "Customer not found"}), 404
    db.execute("DELETE FROM orders WHERE o_custkey = ?", (c_custkey,))
    db.execute("DELETE FROM customer WHERE c_custkey = ?", (c_custkey,))
    db.commit()
    return "", 204


@app.route("/api/orders/monthly-trend", methods=["GET"])
def monthly_trend():
    """
    Return the trend of all existed orders per-month.
    """
    db = get_db()
    rows = db.execute(
        "SELECT strftime('%Y-%m', o_orderdate) AS month, "
        "       COUNT(*) AS order_count, "
        "       SUM(o_totalprice) AS revenue "
        "FROM orders "
        "GROUP BY month "
        "ORDER BY month"
    ).fetchall()
    return jsonify([dict(r) for r in rows]), 200


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=int(os.environ.get("PORT", 5000)))
