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


@app.route("/api/products/<int:p_productkey>/discontinue", methods=["POST"])
def discontinue_product(p_productkey):
    db = get_db()
    existing = db.execute(
        "SELECT p_productkey FROM product WHERE p_productkey = ?", (p_productkey,)
    ).fetchone()
    if existing is None:
        return jsonify({"error": "Product not found"}), 404
    db.execute(
        "UPDATE product SET p_status = 'discontinued' WHERE p_productkey = ?",
        (p_productkey,),
    )
    db.commit()
    return jsonify({"p_productkey": p_productkey, "p_status": "discontinued"}), 200


@app.route("/api/products", methods=["GET"])
def list_products():
    """
    List purchasable products
    """
    db = get_db()
    rows = db.execute(
        "SELECT p_productkey, p_name, p_retailprice "
        "FROM product WHERE p_status = 'active' "
        "ORDER BY p_productkey"
    ).fetchall()
    return jsonify([dict(r) for r in rows]), 200


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=int(os.environ.get("PORT", 5000)))
