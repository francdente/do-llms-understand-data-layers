import os
from datetime import date
from flask import Flask, jsonify
from sqlalchemy import create_engine, Column, Integer, Real, Text, ForeignKey
from sqlalchemy.orm import declarative_base, sessionmaker, scoped_session

app = Flask(__name__)
DATABASE = os.environ.get("DB_PATH", "app.db")

engine = create_engine(f"sqlite:///{DATABASE}")
SessionLocal = scoped_session(sessionmaker(bind=engine))
Base = declarative_base()


class Order(Base):
    __tablename__ = "orders"

    o_orderkey = Column(Integer, primary_key=True)
    o_totalprice = Column(Real)
    o_orderdate = Column(Text)
    o_latest_shipdate = Column(Text)
    o_comment = Column(Text)


class LineItem(Base):
    __tablename__ = "lineitem"

    l_orderkey = Column(Integer, ForeignKey("orders.o_orderkey"), primary_key=True)
    l_linenumber = Column(Integer, primary_key=True)
    l_extendedprice = Column(Real)
    l_discount = Column(Real)
    l_tax = Column(Real)
    l_shipdate = Column(Text)
    l_comment = Column(Text)


@app.teardown_appcontext
def close_db(exception):
    SessionLocal.remove()


@app.route("/api/orders/<int:o_orderkey>/lineitems/<int:l_linenumber>/ship", methods=["POST"])
def ship_lineitem(o_orderkey, l_linenumber):
    db = SessionLocal()
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
        .update({"l_shipdate": today}, synchronize_session=False)
    )
    db.commit()
    return jsonify({"l_orderkey": o_orderkey, "l_linenumber": l_linenumber,
                     "l_shipdate": today}), 200


@app.route("/api/orders/<int:o_orderkey>", methods=["GET"])
def get_order(o_orderkey):
    db = SessionLocal()
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