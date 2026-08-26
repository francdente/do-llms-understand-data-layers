import os
from flask import Flask, request, jsonify, g
from sqlalchemy import create_engine, Column, Integer, Text, Float, ForeignKey, func
from sqlalchemy.orm import declarative_base, sessionmaker

app = Flask(__name__)
DATABASE = os.environ.get("DB_PATH", "app.db")
Base = declarative_base()
engine = create_engine(f"sqlite:///{DATABASE}")
SessionLocal = sessionmaker(bind=engine)


class Order(Base):
    __tablename__ = "orders"

    o_orderkey = Column(Integer, primary_key=True)
    o_orderstatus = Column(Text)
    o_orderdate = Column(Text)
    o_comment = Column(Text)


class LineItem(Base):
    __tablename__ = "lineitem"

    l_orderkey = Column(Integer, ForeignKey("orders.o_orderkey"), primary_key=True)
    l_linenumber = Column(Integer, primary_key=True)
    l_extendedprice = Column(Float)
    l_discount = Column(Float)
    l_tax = Column(Float)
    l_shipdate = Column(Text)
    l_comment = Column(Text)


def get_db():
    db = getattr(g, "_database", None)
    if db is None:
        db = g._database = SessionLocal()
    return db


@app.teardown_appcontext
def close_db(exception):
    db = getattr(g, "_database", None)
    if db is not None:
        db.close()


@app.route("/api/orders/<int:o_orderkey>", methods=["GET"])
def get_order(o_orderkey):
    db = get_db()
    trans = db.begin()
    row = db.query(Order).with_entities(
        Order.o_orderkey,
        Order.o_orderstatus,
        Order.o_orderdate,
        Order.o_comment,
    ).filter(Order.o_orderkey == o_orderkey).first()
    if row is None:
        trans.rollback()
        return jsonify({"error": "Order not found"}), 404
    total = db.query(
        func.coalesce(func.sum(LineItem.l_extendedprice * (1 - LineItem.l_discount) * (1 + LineItem.l_tax)), 0).label("total")
    ).filter(LineItem.l_orderkey == o_orderkey).first()
    trans.commit()
    result = {
        "o_orderkey": row.o_orderkey,
        "o_orderstatus": row.o_orderstatus,
        "o_orderdate": row.o_orderdate,
        "o_comment": row.o_comment,
    }
    result["o_totalprice"] = total.total
    return jsonify(result), 200


@app.route("/api/orders/<int:o_orderkey>/lineitems/<int:l_linenumber>", methods=["PUT"])
def update_lineitem(o_orderkey, l_linenumber):
    payload = request.get_json(silent=True) or {}
    db = get_db()
    trans = db.begin()
    existing = db.query(LineItem).with_entities(
        LineItem.l_orderkey
    ).filter(
        LineItem.l_orderkey == o_orderkey,
        LineItem.l_linenumber == l_linenumber,
    ).first()
    if existing is None:
        trans.rollback()
        return jsonify({"error": "Line item not found"}), 404
    db.query(LineItem).filter(
        LineItem.l_orderkey == o_orderkey,
        LineItem.l_linenumber == l_linenumber,
    ).update({
        LineItem.l_extendedprice: payload["l_extendedprice"],
        LineItem.l_discount: payload["l_discount"],
        LineItem.l_tax: payload["l_tax"],
    }, synchronize_session=False)
    row = db.query(LineItem).with_entities(
        LineItem.l_orderkey,
        LineItem.l_linenumber,
        LineItem.l_extendedprice,
        LineItem.l_discount,
        LineItem.l_tax,
    ).filter(
        LineItem.l_orderkey == o_orderkey,
        LineItem.l_linenumber == l_linenumber,
    ).first()
    db.commit()
    return jsonify({
        "l_orderkey": row.l_orderkey,
        "l_linenumber": row.l_linenumber,
        "l_extendedprice": row.l_extendedprice,
        "l_discount": row.l_discount,
        "l_tax": row.l_tax,
    }), 200

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=int(os.environ.get("PORT", 5000)))