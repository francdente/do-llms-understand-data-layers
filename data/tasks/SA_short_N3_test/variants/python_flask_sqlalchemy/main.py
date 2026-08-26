import os
from datetime import date
from flask import Flask, jsonify, g
from sqlalchemy import create_engine, Column, Integer, Float, Text, ForeignKey, func
from sqlalchemy.orm import declarative_base, sessionmaker

app = Flask(__name__)
DATABASE = os.environ.get("DB_PATH", "app.db")

Base = declarative_base()


class Order(Base):
    __tablename__ = "orders"

    o_orderkey = Column(Integer, primary_key=True)
    o_totalprice = Column(Float)
    o_orderdate = Column(Text)
    o_latest_shipdate = Column(Text)
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


engine = create_engine(f"sqlite:///{DATABASE}")
SessionLocal = sessionmaker(bind=engine)


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


@app.route("/api/orders/<int:o_orderkey>/lineitems/<int:l_linenumber>/ship", methods=["POST"])
def ship_lineitem(o_orderkey, l_linenumber):
    db = get_db()
    existing = (
        db.query(LineItem.l_orderkey)
        .filter(
            LineItem.l_orderkey == o_orderkey,
            LineItem.l_linenumber == l_linenumber,
        )
        .first()
    )
    if existing is None:
        return jsonify({"error": "Line item not found"}), 404
    today = date.today().isoformat()
    (
        db.query(LineItem)
        .filter(
            LineItem.l_orderkey == o_orderkey,
            LineItem.l_linenumber == l_linenumber,
        )
        .update({LineItem.l_shipdate: today}, synchronize_session=False)
    )
    latest = (
        db.query(func.max(LineItem.l_shipdate).label("latest"))
        .filter(LineItem.l_orderkey == o_orderkey)
        .scalar()
    )
    (
        db.query(Order)
        .filter(Order.o_orderkey == o_orderkey)
        .update({Order.o_latest_shipdate: latest}, synchronize_session=False)
    )
    db.commit()
    return jsonify({"l_orderkey": o_orderkey, "l_linenumber": l_linenumber,
                     "l_shipdate": today}), 200


@app.route("/api/orders/<int:o_orderkey>", methods=["GET"])
def get_order(o_orderkey):
    db = get_db()
    row = (
        db.query(
            Order.o_orderkey,
            Order.o_totalprice,
            Order.o_orderdate,
            Order.o_latest_shipdate,
            Order.o_comment,
        )
        .filter(Order.o_orderkey == o_orderkey)
        .first()
    )
    if row is None:
        return jsonify({"error": "Order not found"}), 404
    return jsonify({
        "o_orderkey": row.o_orderkey,
        "o_totalprice": row.o_totalprice,
        "o_orderdate": row.o_orderdate,
        "o_latest_shipdate": row.o_latest_shipdate,
        "o_comment": row.o_comment,
    }), 200


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=int(os.environ.get("PORT", 5000)))