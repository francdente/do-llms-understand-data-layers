import os
from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse
from sqlalchemy import create_engine, Column, Integer, Text, Float, ForeignKey, func
from sqlalchemy.orm import declarative_base, sessionmaker, Session

app = FastAPI()
DATABASE = os.environ.get("DB_PATH", "app.db")

engine = create_engine(
    f"sqlite:///{DATABASE}",
    connect_args={"check_same_thread": False},
    future=True,
)
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
def get_order(o_orderkey: int):
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


@app.post("/api/orders/{o_orderkey}/lineitems")
async def add_lineitem(o_orderkey: int, request: Request):
    payload = await request.json() if request.headers.get("content-type", "").lower().startswith("application/json") else {}
    payload = payload or {}
    db = get_db()
    try:
        db.begin()
        order = (
            db.query(Orders.o_orderkey)
            .filter(Orders.o_orderkey == o_orderkey)
            .first()
        )
        if order is None:
            db.rollback()
            return JSONResponse({"error": "Order not found"}, status_code=404)
        db.add(
            Lineitem(
                l_orderkey=o_orderkey,
                l_linenumber=payload["l_linenumber"],
                l_extendedprice=payload["l_extendedprice"],
                l_discount=payload["l_discount"],
                l_tax=payload["l_tax"],
                l_shipdate=payload.get("l_shipdate"),
                l_comment=payload.get("l_comment", ""),
            )
        )
        db.commit()
        return JSONResponse(
            {"l_orderkey": o_orderkey, "l_linenumber": payload["l_linenumber"]},
            status_code=201,
        )
    finally:
        db.close()


@app.post("/api/orders")
async def create_order(request: Request):
    payload = await request.json() if request.headers.get("content-type", "").lower().startswith("application/json") else {}
    payload = payload or {}
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

        db.query(Orders).filter(Orders.o_orderkey == o_orderkey).update(
            {Orders.o_totalprice: total},
            synchronize_session=False,
        )
        db.commit()
        return JSONResponse(
            {"o_orderkey": o_orderkey, "o_totalprice": total},
            status_code=201,
        )
    finally:
        db.close()


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=int(os.environ.get("PORT", 5000)))