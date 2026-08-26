import os
from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse
from sqlalchemy import create_engine, Column, Integer, Text, Float, ForeignKey, func
from sqlalchemy.orm import declarative_base, sessionmaker, Session

app = FastAPI()
DATABASE = os.environ.get("DB_PATH", "app.db")

engine = create_engine(f"sqlite:///{DATABASE}", future=True)
SessionLocal = sessionmaker(bind=engine, autoflush=False, autocommit=False, future=True)
Base = declarative_base()


class Orders(Base):
    __tablename__ = "orders"

    o_orderkey = Column(Integer, primary_key=True)
    o_orderstatus = Column(Text)
    o_totalprice = Column(Float)
    o_orderdate = Column(Text)
    o_comment = Column(Text)


class Lineitem(Base):
    __tablename__ = "lineitem"

    l_orderkey = Column(Integer, ForeignKey("orders.o_orderkey"), primary_key=True)
    l_linenumber = Column(Integer, primary_key=True)
    l_extendedprice = Column(Float)
    l_discount = Column(Float)
    l_tax = Column(Float)
    l_shipdate = Column(Text)
    l_comment = Column(Text)


def get_db():
    return SessionLocal()


@app.get("/api/orders/{o_orderkey}")
async def get_order(o_orderkey: int):
    db = get_db()
    try:
        row = (
            db.query(
                Orders.o_orderkey,
                Orders.o_orderstatus,
                Orders.o_totalprice,
                Orders.o_orderdate,
                Orders.o_comment,
            )
            .filter(Orders.o_orderkey == o_orderkey)
            .first()
        )
        if row is None:
            return JSONResponse({"error": "Order not found"}, status_code=404)
        return JSONResponse(
            {
                "o_orderkey": row.o_orderkey,
                "o_orderstatus": row.o_orderstatus,
                "o_totalprice": row.o_totalprice,
                "o_orderdate": row.o_orderdate,
                "o_comment": row.o_comment,
            },
            status_code=200,
        )
    finally:
        db.close()


@app.put("/api/orders/{o_orderkey}/lineitems/{l_linenumber}")
async def update_lineitem(o_orderkey: int, l_linenumber: int, request: Request):
    payload = await request.json() if request.headers.get("content-type", "").lower().startswith("application/json") else {}
    db = get_db()
    try:
        db.begin()
        existing = (
            db.query(Lineitem.l_orderkey)
            .filter(
                Lineitem.l_orderkey == o_orderkey,
                Lineitem.l_linenumber == l_linenumber,
            )
            .first()
        )
        if existing is None:
            db.rollback()
            return JSONResponse({"error": "Line item not found"}, status_code=404)
        (
            db.query(Lineitem)
            .filter(
                Lineitem.l_orderkey == o_orderkey,
                Lineitem.l_linenumber == l_linenumber,
            )
            .update(
                {
                    Lineitem.l_extendedprice: payload["l_extendedprice"],
                    Lineitem.l_discount: payload["l_discount"],
                    Lineitem.l_tax: payload["l_tax"],
                },
                synchronize_session=False,
            )
        )
        row = (
            db.query(
                Lineitem.l_orderkey,
                Lineitem.l_linenumber,
                Lineitem.l_extendedprice,
                Lineitem.l_discount,
                Lineitem.l_tax,
            )
            .filter(
                Lineitem.l_orderkey == o_orderkey,
                Lineitem.l_linenumber == l_linenumber,
            )
            .first()
        )
        db.commit()
        return JSONResponse(
            {
                "l_orderkey": row.l_orderkey,
                "l_linenumber": row.l_linenumber,
                "l_extendedprice": row.l_extendedprice,
                "l_discount": row.l_discount,
                "l_tax": row.l_tax,
            },
            status_code=200,
        )
    finally:
        db.close()


@app.post("/api/orders")
async def create_order(request: Request):
    payload = await request.json() if request.headers.get("content-type", "").lower().startswith("application/json") else {}
    db = get_db()
    try:
        db.begin()
        order = Orders(
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
                Lineitem(
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
        total = (
            db.query(
                func.coalesce(
                    func.sum(
                        Lineitem.l_extendedprice
                        * (1 - Lineitem.l_discount)
                        * (1 + Lineitem.l_tax)
                    ),
                    0,
                ).label("total")
            )
            .filter(Lineitem.l_orderkey == o_orderkey)
            .first()
            .total
        )
        (
            db.query(Orders)
            .filter(Orders.o_orderkey == o_orderkey)
            .update({Orders.o_totalprice: total}, synchronize_session=False)
        )
        db.commit()
        return JSONResponse({"o_orderkey": o_orderkey, "o_totalprice": total}, status_code=201)
    finally:
        db.close()


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=int(os.environ.get("PORT", 5000)))