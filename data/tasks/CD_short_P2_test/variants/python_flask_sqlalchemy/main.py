import os
from flask import Flask, jsonify, g
from sqlalchemy import create_engine, Column, Integer, Text, Float, ForeignKey
from sqlalchemy.orm import declarative_base, relationship, sessionmaker, scoped_session

app = Flask(__name__)
DATABASE = os.environ.get("DB_PATH", "app.db")

Base = declarative_base()


class Customer(Base):
    __tablename__ = "customer"

    c_custkey = Column(Integer, primary_key=True)
    c_name = Column(Text)
    c_email = Column(Text)

    orders = relationship("Order", back_populates="customer")


class Order(Base):
    __tablename__ = "orders"

    o_orderkey = Column(Integer, primary_key=True)
    o_custkey = Column(Integer, ForeignKey("customer.c_custkey"))
    o_totalprice = Column(Float)
    o_orderdate = Column(Text)
    o_comment = Column(Text)

    customer = relationship("Customer", back_populates="orders")


engine = create_engine(f"sqlite:///{DATABASE}")
SessionLocal = scoped_session(sessionmaker(bind=engine))


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
    SessionLocal.remove()


@app.route("/api/customers/<int:c_custkey>", methods=["DELETE"])
def delete_customer(c_custkey):
    db = get_db()
    existing = db.query(Customer.c_custkey).filter(Customer.c_custkey == c_custkey).first()
    if existing is None:
        return jsonify({"error": "Customer not found"}), 404
    db.query(Order).filter(Order.o_custkey == c_custkey).delete(synchronize_session=False)
    db.query(Customer).filter(Customer.c_custkey == c_custkey).delete(synchronize_session=False)
    db.commit()
    return "", 204


@app.route("/api/orders/history", methods=["GET"])
def orders_history():
    """
    Return the history of all existed orders
    """
    db = get_db()
    rows = (
        db.query(
            Order.o_orderkey,
            Order.o_custkey,
            Order.o_totalprice,
            Order.o_orderdate,
            Order.o_comment,
        )
        .order_by(Order.o_orderdate)
        .all()
    )
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