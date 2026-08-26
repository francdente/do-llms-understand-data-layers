import os
from flask import Flask, jsonify, g
from sqlalchemy import create_engine, Column, Integer, Text, Float, ForeignKey, func
from sqlalchemy.exc import IntegrityError
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


engine = create_engine(
    f"sqlite:///{DATABASE}",
    connect_args={"check_same_thread": False},
)
SessionLocal = scoped_session(sessionmaker(bind=engine))


def get_db():
    db = getattr(g, "_database", None)
    if db is None:
        session = SessionLocal()
        session.execute(func.sqlite_version().select() if False else "PRAGMA foreign_keys = ON;")
        g._database = session
        db = session
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
    try:
        db.query(Customer).filter(Customer.c_custkey == c_custkey).delete(synchronize_session=False)
        db.commit()
    except IntegrityError:
        db.rollback()
        return jsonify({"error": "Cannot delete customer with existing orders"}), 409
    return "", 204


@app.route("/api/dashboard", methods=["GET"])
def dashboard():
    """
    Return dashboard based on all existed orders
    """
    db = get_db()
    row = db.query(
        func.count().label("total_orders"),
        func.coalesce(func.sum(Order.o_totalprice), 0).label("total_revenue"),
    ).select_from(Order).one()
    return jsonify(
        {
            "total_orders": row.total_orders,
            "total_revenue": row.total_revenue,
        }
    ), 200


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=int(os.environ.get("PORT", 5000)))