import os
from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.responses import JSONResponse, Response
from sqlalchemy import Column, Float, ForeignKey, Integer, Text, create_engine, func
from sqlalchemy.orm import declarative_base, sessionmaker

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


@asynccontextmanager
async def lifespan(app: FastAPI):
    yield
    engine.dispose()


app = FastAPI(lifespan=lifespan)


@app.delete("/api/customers/{c_custkey}")
def delete_customer(c_custkey: int):
    db = SessionLocal()
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


@app.get("/api/orders/summary")
def orders_summary():
    """
    Return the summary of all existed historical orders
    """
    db = SessionLocal()
    try:
        row = db.query(
            func.count().label("total_orders"),
            func.coalesce(func.sum(Order.o_totalprice), 0).label("total_revenue"),
        ).select_from(Order).one()
        return JSONResponse(
            {
                "total_orders": row.total_orders,
                "total_revenue": row.total_revenue,
            },
            status_code=200,
        )
    finally:
        db.close()


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=int(os.environ.get("PORT", 5000)))