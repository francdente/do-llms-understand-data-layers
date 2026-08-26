import os
from fastapi import FastAPI, Depends
from fastapi.responses import JSONResponse, Response
from sqlalchemy import create_engine, Column, Integer, Text, Float, ForeignKey, select, delete
from sqlalchemy.orm import declarative_base, sessionmaker, Session

DATABASE = os.environ.get("DB_PATH", "app.db")
DATABASE_URL = f"sqlite:///{DATABASE}"

engine = create_engine(DATABASE_URL, connect_args={"check_same_thread": False})
SessionLocal = sessionmaker(bind=engine, autoflush=False, autocommit=False)

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


app = FastAPI()


def get_db():
    db = SessionLocal()
    try:
        yield db
    finally:
        db.close()


@app.delete("/api/customers/{c_custkey}")
def delete_customer(c_custkey: int, db: Session = Depends(get_db)):
    existing = db.execute(
        select(Customer.c_custkey).where(Customer.c_custkey == c_custkey)
    ).first()
    if existing is None:
        return JSONResponse({"error": "Customer not found"}, status_code=404)
    db.execute(delete(Order).where(Order.o_custkey == c_custkey))
    db.execute(delete(Customer).where(Customer.c_custkey == c_custkey))
    db.commit()
    return Response(status_code=204)


@app.get("/api/orders")
def list_orders(db: Session = Depends(get_db)):
    """
    Return orders tied to active customers
    """
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


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=int(os.environ.get("PORT", 5000)))