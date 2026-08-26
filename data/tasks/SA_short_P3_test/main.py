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


@app.route("/api/orders/<int:o_orderkey>/lineitems/<int:l_linenumber>/ship", methods=["POST"])
def ship_lineitem(o_orderkey, l_linenumber):
    db = get_db()
    db.execute("BEGIN")
    existing = db.execute(
        "SELECT l_orderkey FROM lineitem WHERE l_orderkey = ? AND l_linenumber = ?",
        (o_orderkey, l_linenumber),
    ).fetchone()
    if existing is None:
        db.execute("ROLLBACK")
        return jsonify({"error": "Line item not found"}), 404
    payload = request.get_json(silent=True) or {}
    shipdate = payload["l_shipdate"]
    db.execute(
        "UPDATE lineitem SET l_shipdate = ? "
        "WHERE l_orderkey = ? AND l_linenumber = ?",
        (shipdate, o_orderkey, l_linenumber),
    )
    db.commit()
    return jsonify({"l_orderkey": o_orderkey, "l_linenumber": l_linenumber,
                     "l_shipdate": shipdate}), 200


@app.route("/api/orders/<int:o_orderkey>", methods=["GET"])
def get_order(o_orderkey):
    db = get_db()
    row = db.execute(
        "SELECT o_orderkey, o_totalprice, o_orderdate, o_latest_shipdate, o_comment "
        "FROM orders WHERE o_orderkey = ?",
        (o_orderkey,),
    ).fetchone()
    if row is None:
        return jsonify({"error": "Order not found"}), 404
    return jsonify(dict(row)), 200


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=int(os.environ.get("PORT", 5000)))
