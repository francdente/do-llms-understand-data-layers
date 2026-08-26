import os
from flask import Flask, jsonify
from sqlalchemy import create_engine, Column, Integer, Text, Float, ForeignKey, func
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
    existing = db.query(Customer.c_custkey).filter(Customer.c_custkey == c_custkey).first()
    if existing is None:
        return jsonify({"error": "Customer not found"}), 404
    db.query(Order).filter(Order.o_custkey == c_custkey).delete(synchronize_session=False)
    db.query(Customer).filter(Customer.c_custkey == c_custkey).delete(synchronize_session=False)
    db.commit()
    return "", 204


@app.route("/api/orders/summary", methods=["GET"])
def orders_summary():
    """
    Return the summary of all existed historical orders
    """
    db = SessionLocal()
    row = db.query(
        func.count().label("total_orders"),
        func.coalesce(func.sum(Order.o_totalprice), 0).label("total_revenue"),
    ).select_from(Order).first()
    return jsonify(
        {
            "total_orders": row.total_orders,
            "total_revenue": row.total_revenue,
        }
    ), 200


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=int(os.environ.get("PORT", 5000)))