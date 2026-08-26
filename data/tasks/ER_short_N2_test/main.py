import os
import sqlite3
from flask import Flask, request, jsonify, g

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


@app.route("/api/products/<int:p_productkey>/price", methods=["PUT"])
def update_product_price(p_productkey):
    payload = request.get_json(silent=True) or {}
    db = get_db()
    existing = db.execute(
        "SELECT p_productkey FROM product WHERE p_productkey = ?", (p_productkey,)
    ).fetchone()
    if existing is None:
        return jsonify({"error": "Product not found"}), 404
    db.execute(
        "UPDATE product SET p_retailprice = ? WHERE p_productkey = ?",
        (payload["p_retailprice"], p_productkey),
    )
    db.commit()
    row = db.execute(
        "SELECT p_productkey, p_name, p_status, p_retailprice "
        "FROM product WHERE p_productkey = ?",
        (p_productkey,),
    ).fetchone()
    return jsonify(dict(row)), 200


@app.route("/api/products/<int:p_productkey>", methods=["GET"])
def get_product(p_productkey):
    db = get_db()
    row = db.execute(
        "SELECT p_productkey, p_name, p_status, p_retailprice "
        "FROM product WHERE p_productkey = ?",
        (p_productkey,),
    ).fetchone()
    if row is None:
        return jsonify({"error": "Product not found"}), 404
    return jsonify(dict(row)), 200


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=int(os.environ.get("PORT", 5000)))
