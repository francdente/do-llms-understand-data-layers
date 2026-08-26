import os
from fastapi import FastAPI, Depends, Response
from fastapi.responses import JSONResponse
from sqlalchemy import create_engine, Column, Integer, Text, Float, ForeignKey, func
from sqlalchemy.orm import declarative_base, sessionmaker, Session

app = FastAPI()
DATABASE = os.environ.get("DB_PATH", "app.db")

engine = create_engine(f"sqlite:///{DATABASE}", connect_args={"check_same_thread": False})
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


def get_db():
    db = SessionLocal()
    try:
        yield db
    finally:
        db.close()


@app.delete("/api/customers/{c_custkey}")
def delete_customer(c_custkey: int, db: Session = Depends(get_db)):
    existing = db.query(Customer.c_custkey).filter(Customer.c_custkey == c_custkey).first()
    if existing is None:
        return JSONResponse(content={"error": "Customer not found"}, status_code=404)
    db.query(Order).filter(Order.o_custkey == c_custkey).delete(synchronize_session=False)
    db.query(Customer).filter(Customer.c_custkey == c_custkey).delete(synchronize_session=False)
    db.commit()
    return Response(status_code=204)


@app.get("/api/orders/monthly-trend")
def monthly_trend(db: Session = Depends(get_db)):
    """
    Return the trend of all existed orders per-month.
    """
    rows = (
        db.query(
            func.strftime("%Y-%m", Order.o_orderdate).label("month"),
            func.count().label("order_count"),
            func.sum(Order.o_totalprice).label("revenue"),
        )
        .group_by("month")
        .order_by("month")
        .all()
    )
    return [{"month": r.month, "order_count": r.order_count, "revenue": r.revenue} for r in rows]


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=int(os.environ.get("PORT", 5000)))