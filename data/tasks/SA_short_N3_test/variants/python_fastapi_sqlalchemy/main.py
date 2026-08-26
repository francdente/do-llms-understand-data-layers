import os
from datetime import date

from fastapi import FastAPI, Depends
from fastapi.responses import JSONResponse
from sqlalchemy import create_engine, Column, Integer, Float, Text, ForeignKey, func
from sqlalchemy.orm import declarative_base, sessionmaker, Session

app = FastAPI()
DATABASE = os.environ.get("DB_PATH", "app.db")

engine = create_engine(f"sqlite:///{DATABASE}", connect_args={"check_same_thread": False})
SessionLocal = sessionmaker(bind=engine, autoflush=False, autocommit=False)

Base = declarative_base()


class Order(Base):
    __tablename__ = "orders"

    o_orderkey = Column(Integer, primary_key=True)
    o_totalprice = Column(Float)
    o_orderdate = Column(Text)
    o_latest_shipdate = Column(Text)
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
    db = SessionLocal()
    try:
        yield db
    finally:
        db.close()


@app.post("/api/orders/{o_orderkey}/lineitems/{l_linenumber}/ship")
def ship_lineitem(o_orderkey: int, l_linenumber: int, db: Session = Depends(get_db)):
    existing = (
        db.query(LineItem.l_orderkey)
        .filter(
            LineItem.l_orderkey == o_orderkey,
            LineItem.l_linenumber == l_linenumber,
        )
        .first()
    )
    if existing is None:
        return JSONResponse({"error": "Line item not found"}, status_code=404)
    today = date.today().isoformat()
    (
        db.query(LineItem)
        .filter(
            LineItem.l_orderkey == o_orderkey,
            LineItem.l_linenumber == l_linenumber,
        )
        .update({"l_shipdate": today}, synchronize_session=False)
    )
    latest = (
        db.query(func.max(LineItem.l_shipdate).label("latest"))
        .filter(LineItem.l_orderkey == o_orderkey)
        .scalar()
    )
    (
        db.query(Order)
        .filter(Order.o_orderkey == o_orderkey)
        .update({"o_latest_shipdate": latest}, synchronize_session=False)
    )
    db.commit()
    return JSONResponse(
        {
            "l_orderkey": o_orderkey,
            "l_linenumber": l_linenumber,
            "l_shipdate": today,
        },
        status_code=200,
    )


@app.get("/api/orders/{o_orderkey}")
def get_order(o_orderkey: int, db: Session = Depends(get_db)):
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
        return JSONResponse({"error": "Order not found"}, status_code=404)
    return JSONResponse(
        {
            "o_orderkey": row.o_orderkey,
            "o_totalprice": row.o_totalprice,
            "o_orderdate": row.o_orderdate,
            "o_latest_shipdate": row.o_latest_shipdate,
            "o_comment": row.o_comment,
        },
        status_code=200,
    )


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=int(os.environ.get("PORT", 5000)))