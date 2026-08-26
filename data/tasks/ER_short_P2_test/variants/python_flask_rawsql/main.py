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


@app.route("/api/products/<int:p_productkey>/availability", methods=["GET"])
def get_product_availability(p_productkey):
    """
    Get availability for purchasable product
    """
    db = get_db()
    row = db.execute(
        "SELECT p.p_productkey, p.p_name, "
        "       COALESCE(SUM(i.i_quantity), 0) AS total_stock "
        "FROM product p LEFT JOIN inventory i ON p.p_productkey = i.i_productkey "
        "WHERE p.p_productkey = ? "
        "GROUP BY p.p_productkey",
        (p_productkey,),
    ).fetchone()
    if row is None:
        return jsonify({"error": "Product not found"}), 404
    return jsonify(dict(row)), 200


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=int(os.environ.get("PORT", 5000)))
