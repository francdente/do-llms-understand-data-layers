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


@app.route("/api/orders/<int:o_orderkey>", methods=["GET"])
def get_order(o_orderkey):
    db = get_db()
    row = db.execute(
        "SELECT o_orderkey, o_orderstatus, o_totalprice, o_orderdate, o_comment "
        "FROM orders WHERE o_orderkey = ?",
        (o_orderkey,),
    ).fetchone()
    if row is None:
        return jsonify({"error": "Order not found"}), 404
    return jsonify(dict(row)), 200


@app.route("/api/orders/<int:o_orderkey>/lineitems", methods=["POST"])
def add_lineitem(o_orderkey):
    payload = request.get_json(silent=True) or {}
    db = get_db()
    db.execute("BEGIN")
    order = db.execute(
        "SELECT o_orderkey FROM orders WHERE o_orderkey = ?", (o_orderkey,)
    ).fetchone()
    if order is None:
        db.execute("ROLLBACK")
        return jsonify({"error": "Order not found"}), 404
    db.execute(
        "INSERT INTO lineitem (l_orderkey, l_linenumber, l_extendedprice, l_discount, l_tax, l_shipdate, l_comment) "
        "VALUES (?, ?, ?, ?, ?, ?, ?)",
        (o_orderkey, payload["l_linenumber"], payload["l_extendedprice"],
         payload["l_discount"], payload["l_tax"],
         payload.get("l_shipdate"), payload.get("l_comment", "")),
    )
    db.commit()
    return jsonify({"l_orderkey": o_orderkey, "l_linenumber": payload["l_linenumber"]}), 201


@app.route("/api/orders", methods=["POST"])
def create_order():
    payload = request.get_json(silent=True) or {}
    db = get_db()
    db.execute("BEGIN")
    cur = db.execute(
        "INSERT INTO orders (o_orderstatus, o_totalprice, o_orderdate, o_comment) "
        "VALUES (?, 0, ?, ?)",
        (payload.get("o_orderstatus", "O"), payload["o_orderdate"],
         payload.get("o_comment", "")),
    )
    o_orderkey = cur.lastrowid
    for item in payload["lineitems"]:
        db.execute(
            "INSERT INTO lineitem (l_orderkey, l_linenumber, l_extendedprice, "
            "l_discount, l_tax, l_shipdate, l_comment) "
            "VALUES (?, ?, ?, ?, ?, ?, ?)",
            (o_orderkey, item["l_linenumber"], item["l_extendedprice"],
             item["l_discount"], item["l_tax"],
             item.get("l_shipdate"), item.get("l_comment", "")),
        )
    total = db.execute(
        "SELECT COALESCE(SUM(l_extendedprice * (1 - l_discount) * (1 + l_tax)), 0) AS total "
        "FROM lineitem WHERE l_orderkey = ?",
        (o_orderkey,),
    ).fetchone()["total"]
    db.execute(
        "UPDATE orders SET o_totalprice = ? WHERE o_orderkey = ?",
        (total, o_orderkey),
    )
    db.commit()
    return jsonify({"o_orderkey": o_orderkey, "o_totalprice": total}), 201


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=int(os.environ.get("PORT", 5000)))
