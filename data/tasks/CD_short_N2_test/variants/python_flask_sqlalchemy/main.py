import os
from flask import Flask, jsonify
from sqlalchemy import create_engine, Column, Integer, Text, Float, ForeignKey, select
from sqlalchemy.orm import declarative_base, sessionmaker, scoped_session

app = Flask(__name__)
DATABASE = os.environ.get("DB_PATH", "app.db")

engine = create_engine(f"sqlite:///{DATABASE}")
SessionLocal = scoped_session(sessionmaker(bind=engine))
Base = declarative_base()


class Customer(Base):
    __tablename__ = "customer"

    c_custkey = Column(Integer, primary_key=True)
    c_name = Column(Text)
    c_email = Column(Text)


class Order(Base):
    __tablename__ = "orders"

    o_orderkey = Column(Integer, primary_key=True)
    o_custkey = Column(Integer, ForeignKey("customer.c_custkey"))
    o_totalprice = Column(Float)
    o_orderdate = Column(Text)
    o_comment = Column(Text)


@app.teardown_appcontext
def close_db(exception):
    SessionLocal.remove()


@app.route("/api/customers/<int:c_custkey>", methods=["DELETE"])
def delete_customer(c_custkey):
    db = SessionLocal()
    existing = db.execute(
        select(Customer.c_custkey).where(Customer.c_custkey == c_custkey)
    ).first()
    if existing is None:
        return jsonify({"error": "Customer not found"}), 404
    db.query(Order).filter(Order.o_custkey == c_custkey).delete(synchronize_session=False)
    db.query(Customer).filter(Customer.c_custkey == c_custkey).delete(synchronize_session=False)
    db.commit()
    return "", 204


@app.route("/api/orders", methods=["GET"])
def list_orders():
    """
    Return orders tied to active customers
    """
    db = SessionLocal()
    subquery = select(Customer.c_custkey)
    rows = db.execute(
        select(
            Order.o_orderkey,
            Order.o_custkey,
            Order.o_totalprice,
            Order.o_orderdate,
            Order.o_comment,
        )
        .where(Order.o_custkey.in_(subquery))
        .order_by(Order.o_orderdate)
    ).all()
    return jsonify(
        [
            {
                "o_orderkey": r.o_orderkey,
                "o_custkey": r.o_custkey,
                "o_totalprice": r.o_totalprice,
                "o_orderdate": r.o_orderdate,
                "o_comment": r.o_comment,
            }
            for r in rows
        ]
    ), 200


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=int(os.environ.get("PORT", 5000)))