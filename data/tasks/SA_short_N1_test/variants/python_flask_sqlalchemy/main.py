import os
from flask import Flask, request, jsonify, g
from sqlalchemy import create_engine, Column, Integer, Text, Float, ForeignKey, func
from sqlalchemy.orm import declarative_base, sessionmaker

app = Flask(__name__)
DATABASE = os.environ.get("DB_PATH", "app.db")

engine = create_engine(f"sqlite:///{DATABASE}")
SessionLocal = sessionmaker(bind=engine, autoflush=True, autocommit=False)

Base = declarative_base()


class Order(Base):
    __tablename__ = "orders"

    o_orderkey = Column(Integer, primary_key=True)
    o_orderstatus = Column(Text)
    o_totalprice = Column(Float)
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
    row = (
        db.query(
            Order.o_orderkey,
            Order.o_orderstatus,
            Order.o_totalprice,
            Order.o_orderdate,
            Order.o_comment,
        )
        .filter(Order.o_orderkey == o_orderkey)
        .first()
    )
    if row is None:
        return jsonify({"error": "Order not found"}), 404
    return jsonify(
        {
            "o_orderkey": row.o_orderkey,
            "o_orderstatus": row.o_orderstatus,
            "o_totalprice": row.o_totalprice,
            "o_orderdate": row.o_orderdate,
            "o_comment": row.o_comment,
        }
    ), 200


@app.route("/api/orders/<int:o_orderkey>/lineitems/<int:l_linenumber>", methods=["PUT"])
def update_lineitem(o_orderkey, l_linenumber):
    payload = request.get_json(silent=True) or {}
    db = get_db()
    try:
        existing = (
            db.query(LineItem.l_orderkey)
            .filter(
                LineItem.l_orderkey == o_orderkey,
                LineItem.l_linenumber == l_linenumber,
            )
            .first()
        )
        if existing is None:
            db.rollback()
            return jsonify({"error": "Line item not found"}), 404

        (
            db.query(LineItem)
            .filter(
                LineItem.l_orderkey == o_orderkey,
                LineItem.l_linenumber == l_linenumber,
            )
            .update(
                {
                    LineItem.l_extendedprice: payload["l_extendedprice"],
                    LineItem.l_discount: payload["l_discount"],
                    LineItem.l_tax: payload["l_tax"],
                },
                synchronize_session=False,
            )
        )

        total_subquery = (
            db.query(
                func.coalesce(
                    func.sum(
                        LineItem.l_extendedprice
                        * (1 - LineItem.l_discount)
                        * (1 + LineItem.l_tax)
                    ),
                    0,
                )
            )
            .filter(LineItem.l_orderkey == o_orderkey)
            .scalar()
        )

        (
            db.query(Order)
            .filter(Order.o_orderkey == o_orderkey)
            .update({Order.o_totalprice: total_subquery}, synchronize_session=False)
        )

        row = (
            db.query(
                LineItem.l_orderkey,
                LineItem.l_linenumber,
                LineItem.l_extendedprice,
                LineItem.l_discount,
                LineItem.l_tax,
            )
            .filter(
                LineItem.l_orderkey == o_orderkey,
                LineItem.l_linenumber == l_linenumber,
            )
            .first()
        )
        db.commit()
        return jsonify(
            {
                "l_orderkey": row.l_orderkey,
                "l_linenumber": row.l_linenumber,
                "l_extendedprice": row.l_extendedprice,
                "l_discount": row.l_discount,
                "l_tax": row.l_tax,
            }
        ), 200
    except Exception:
        db.rollback()
        raise


@app.route("/api/orders", methods=["POST"])
def create_order():
    payload = request.get_json(silent=True) or {}
    db = get_db()
    try:
        order = Order(
            o_orderstatus=payload.get("o_orderstatus", "O"),
            o_totalprice=0,
            o_orderdate=payload["o_orderdate"],
            o_comment=payload.get("o_comment", ""),
        )
        db.add(order)
        db.flush()
        o_orderkey = order.o_orderkey

        for item in payload["lineitems"]:
            db.add(
                LineItem(
                    l_orderkey=o_orderkey,
                    l_linenumber=item["l_linenumber"],
                    l_extendedprice=item["l_extendedprice"],
                    l_discount=item["l_discount"],
                    l_tax=item["l_tax"],
                    l_shipdate=item.get("l_shipdate"),
                    l_comment=item.get("l_comment", ""),
                )
            )

        db.flush()

        total = (
            db.query(
                func.coalesce(
                    func.sum(
                        LineItem.l_extendedprice
                        * (1 - LineItem.l_discount)
                        * (1 + LineItem.l_tax)
                    ),
                    0,
                ).label("total")
            )
            .filter(LineItem.l_orderkey == o_orderkey)
            .scalar()
        )

        (
            db.query(Order)
            .filter(Order.o_orderkey == o_orderkey)
            .update({Order.o_totalprice: total}, synchronize_session=False)
        )

        db.commit()
        return jsonify({"o_orderkey": o_orderkey, "o_totalprice": total}), 201
    except Exception:
        db.rollback()
        raise


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=int(os.environ.get("PORT", 5000)))