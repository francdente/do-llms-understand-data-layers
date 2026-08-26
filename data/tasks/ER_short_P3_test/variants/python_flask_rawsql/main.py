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


@app.route("/api/suppliers/<int:s_suppkey>/suspend", methods=["POST"])
def suspend_supplier(s_suppkey):
    db = get_db()
    existing = db.execute(
        "SELECT s_suppkey FROM supplier WHERE s_suppkey = ?", (s_suppkey,)
    ).fetchone()
    if existing is None:
        return jsonify({"error": "Supplier not found"}), 404
    db.execute(
        "UPDATE supplier SET s_status = 'suspended' WHERE s_suppkey = ?",
        (s_suppkey,),
    )
    db.commit()
    return jsonify({"s_suppkey": s_suppkey, "s_status": "suspended"}), 200


@app.route("/api/products", methods=["GET"])
def list_products():
    """
    List purchasable products
    """
    db = get_db()
    rows = db.execute(
        "SELECT p.p_productkey, p.p_name, p.p_retailprice, s.s_name AS supplier_name "
        "FROM product p JOIN supplier s ON p.p_suppkey = s.s_suppkey "
        "ORDER BY p.p_productkey"
    ).fetchall()
    return jsonify([dict(r) for r in rows]), 200


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=int(os.environ.get("PORT", 5000)))
