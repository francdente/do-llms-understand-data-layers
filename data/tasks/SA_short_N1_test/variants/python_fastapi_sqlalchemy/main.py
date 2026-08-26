import os
from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse
from sqlalchemy import create_engine, Column, Integer, Text, Float, ForeignKey, select, func, update
from sqlalchemy.orm import declarative_base, relationship, sessionmaker, Session

app = FastAPI()
DATABASE = os.environ.get("DB_PATH", "app.db")

engine = create_engine(f"sqlite:///{DATABASE}", future=True)
SessionLocal = sessionmaker(bind=engine, autoflush=False, autocommit=False, future=True)
Base = declarative_base()


class Order(Base):
    __tablename__ = "orders"

    o_orderkey = Column(Integer, primary_key=True)
    o_orderstatus = Column(Text)
    o_totalprice = Column(Float)
    o_orderdate = Column(Text)
    o_comment = Column(Text)

    lineitems = relationship("LineItem", back_populates="order")


class LineItem(Base):
    __tablename__ = "lineitem"

    l_orderkey = Column(Integer, ForeignKey("orders.o_orderkey"), primary_key=True)
    l_linenumber = Column(Integer, primary_key=True)
    l_extendedprice = Column(Float)
    l_discount = Column(Float)
    l_tax = Column(Float)
    l_shipdate = Column(Text)
    l_comment = Column(Text)

    order = relationship("Order", back_populates="lineitems")


def get_db():
    db = SessionLocal()
    try:
        yield db
    finally:
        db.close()


@app.get("/api/orders/{o_orderkey}")
async def get_order(o_orderkey: int):
    db = SessionLocal()
    try:
        row = db.execute(
            select(
                Order.o_orderkey,
                Order.o_orderstatus,
                Order.o_totalprice,
                Order.o_orderdate,
                Order.o_comment,
            ).where(Order.o_orderkey == o_orderkey)
        ).first()
        if row is None:
            return JSONResponse({"error": "Order not found"}, status_code=404)
        data = row._mapping
        return JSONResponse(
            {
                "o_orderkey": data["o_orderkey"],
                "o_orderstatus": data["o_orderstatus"],
                "o_totalprice": data["o_totalprice"],
                "o_orderdate": data["o_orderdate"],
                "o_comment": data["o_comment"],
            },
            status_code=200,
        )
    finally:
        db.close()


@app.put("/api/orders/{o_orderkey}/lineitems/{l_linenumber}")
async def update_lineitem(o_orderkey: int, l_linenumber: int, request: Request):
    payload = await request.json() if request.headers.get("content-type", "").lower().startswith("application/json") else {}
    db = SessionLocal()
    try:
        db.begin()
        existing = db.execute(
            select(LineItem.l_orderkey).where(
                LineItem.l_orderkey == o_orderkey,
                LineItem.l_linenumber == l_linenumber,
            )
        ).first()
        if existing is None:
            db.rollback()
            return JSONResponse({"error": "Line item not found"}, status_code=404)

        db.execute(
            update(LineItem)
            .where(
                LineItem.l_orderkey == o_orderkey,
                LineItem.l_linenumber == l_linenumber,
            )
            .values(
                l_extendedprice=payload["l_extendedprice"],
                l_discount=payload["l_discount"],
                l_tax=payload["l_tax"],
            )
        )

        total_subquery = (
            select(
                func.coalesce(
                    func.sum(
                        LineItem.l_extendedprice * (1 - LineItem.l_discount) * (1 + LineItem.l_tax)
                    ),
                    0,
                )
            )
            .where(LineItem.l_orderkey == o_orderkey)
            .scalar_subquery()
        )

        db.execute(
            update(Order)
            .where(Order.o_orderkey == o_orderkey)
            .values(o_totalprice=total_subquery)
        )

        row = db.execute(
            select(
                LineItem.l_orderkey,
                LineItem.l_linenumber,
                LineItem.l_extendedprice,
                LineItem.l_discount,
                LineItem.l_tax,
            ).where(
                LineItem.l_orderkey == o_orderkey,
                LineItem.l_linenumber == l_linenumber,
            )
        ).first()
        db.commit()
        data = row._mapping
        return JSONResponse(
            {
                "l_orderkey": data["l_orderkey"],
                "l_linenumber": data["l_linenumber"],
                "l_extendedprice": data["l_extendedprice"],
                "l_discount": data["l_discount"],
                "l_tax": data["l_tax"],
            },
            status_code=200,
        )
    finally:
        db.close()


@app.post("/api/orders")
async def create_order(request: Request):
    payload = await request.json() if request.headers.get("content-type", "").lower().startswith("application/json") else {}
    db = SessionLocal()
    try:
        db.begin()
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

        total = db.execute(
            select(
                func.coalesce(
                    func.sum(
                        LineItem.l_extendedprice * (1 - LineItem.l_discount) * (1 + LineItem.l_tax)
                    ),
                    0,
                ).label("total")
            ).where(LineItem.l_orderkey == o_orderkey)
        ).first()._mapping["total"]

        db.execute(
            update(Order)
            .where(Order.o_orderkey == o_orderkey)
            .values(o_totalprice=total)
        )
        db.commit()
        return JSONResponse({"o_orderkey": o_orderkey, "o_totalprice": total}, status_code=201)
    finally:
        db.close()


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=int(os.environ.get("PORT", 5000)))