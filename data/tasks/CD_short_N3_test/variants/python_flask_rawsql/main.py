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
        db.execute("PRAGMA foreign_keys = ON;")
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
    try:
        db.execute("DELETE FROM customer WHERE c_custkey = ?", (c_custkey,))
        db.commit()
    except sqlite3.IntegrityError:
        return jsonify({"error": "Cannot delete customer with existing orders"}), 409
    return "", 204


@app.route("/api/dashboard", methods=["GET"])
def dashboard():
    """
    Return dashboard based on all existed orders
    """
    db = get_db()
    row = db.execute(
        "SELECT COUNT(*) AS total_orders, "
        "       COALESCE(SUM(o_totalprice), 0) AS total_revenue "
        "FROM orders"
    ).fetchone()
    return jsonify(dict(row)), 200


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=int(os.environ.get("PORT", 5000)))
