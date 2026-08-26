import os
from fastapi import FastAPI
from fastapi.responses import JSONResponse, Response
from sqlalchemy import create_engine, Column, Integer, Text, Float, ForeignKey
from sqlalchemy.orm import declarative_base, sessionmaker, Session

app = FastAPI()
DATABASE = os.environ.get("DB_PATH", "app.db")

engine = create_engine(f"sqlite:///{DATABASE}", future=True)
SessionLocal = sessionmaker(bind=engine, autoflush=False, autocommit=False, future=True)
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


def get_db():
    return SessionLocal()


@app.delete("/api/customers/{c_custkey}")
def delete_customer(c_custkey: int):
    db = get_db()
    try:
        existing = db.query(Customer.c_custkey).filter(Customer.c_custkey == c_custkey).first()
        if existing is None:
            return JSONResponse({"error": "Customer not found"}, status_code=404)
        db.query(Order).filter(Order.o_custkey == c_custkey).delete(synchronize_session=False)
        db.query(Customer).filter(Customer.c_custkey == c_custkey).delete(synchronize_session=False)
        db.commit()
        return Response(status_code=204)
    finally:
        db.close()


@app.get("/api/orders")
def list_orders():
    """
    Return orders tied to active customers
    """
    db = get_db()
    try:
        rows = (
            db.query(
                Order.o_orderkey,
                Order.o_custkey,
                Order.o_totalprice,
                Order.o_orderdate,
                Order.o_comment,
            )
            .join(Customer, Order.o_custkey == Customer.c_custkey)
            .order_by(Order.o_orderdate)
            .all()
        )
        return [
            {
                "o_orderkey": row.o_orderkey,
                "o_custkey": row.o_custkey,
                "o_totalprice": row.o_totalprice,
                "o_orderdate": row.o_orderdate,
                "o_comment": row.o_comment,
            }
            for row in rows
        ]
    finally:
        db.close()


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=int(os.environ.get("PORT", 5000)))