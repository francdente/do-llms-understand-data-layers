import os
from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse
from sqlalchemy import create_engine, Column, Integer, Text, Float, ForeignKey, func
from sqlalchemy.orm import declarative_base, sessionmaker

app = FastAPI()
DATABASE = os.environ.get("DB_PATH", "app.db")

engine = create_engine(f"sqlite:///{DATABASE}")
SessionLocal = sessionmaker(bind=engine, autoflush=False, autocommit=False)

Base = declarative_base()


class Order(Base):
    __tablename__ = "orders"

    o_orderkey = Column(Integer, primary_key=True)
    o_orderstatus = Column(Text)
    o_orderdate = Column(Text)
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


@app.get("/api/orders/{o_orderkey}")
def get_order(o_orderkey: int):
    db = SessionLocal()
    transaction = db.begin()
    try:
        row = db.query(Order).filter(Order.o_orderkey == o_orderkey).first()
        if row is None:
            transaction.rollback()
            return JSONResponse({"error": "Order not found"}, status_code=404)
        total = db.query(
            func.coalesce(
                func.sum(
                    LineItem.l_extendedprice * (1 - LineItem.l_discount) * (1 + LineItem.l_tax)
                ),
                0,
            ).label("total")
        ).filter(LineItem.l_orderkey == o_orderkey).first()
        transaction.commit()
        result = {
            "o_orderkey": row.o_orderkey,
            "o_orderstatus": row.o_orderstatus,
            "o_orderdate": row.o_orderdate,
            "o_comment": row.o_comment,
            "o_totalprice": total.total,
        }
        return JSONResponse(result, status_code=200)
    except Exception:
        transaction.rollback()
        raise
    finally:
        db.close()


@app.put("/api/orders/{o_orderkey}/lineitems/{l_linenumber}")
async def update_lineitem(o_orderkey: int, l_linenumber: int, request: Request):
    payload = await request.json() if request.headers.get("content-type", "").startswith("application/json") else {}
    db = SessionLocal()
    transaction = db.begin()
    try:
        existing = db.query(LineItem.l_orderkey).filter(
            LineItem.l_orderkey == o_orderkey,
            LineItem.l_linenumber == l_linenumber,
        ).first()
        if existing is None:
            transaction.rollback()
            return JSONResponse({"error": "Line item not found"}, status_code=404)
        db.query(LineItem).filter(
            LineItem.l_orderkey == o_orderkey,
            LineItem.l_linenumber == l_linenumber,
        ).update(
            {
                LineItem.l_extendedprice: payload["l_extendedprice"],
                LineItem.l_discount: payload["l_discount"],
                LineItem.l_tax: payload["l_tax"],
            },
            synchronize_session=False,
        )
        row = db.query(LineItem).filter(
            LineItem.l_orderkey == o_orderkey,
            LineItem.l_linenumber == l_linenumber,
        ).first()
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
    except Exception:
        transaction.rollback()
        raise
    finally:
        db.close()


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=int(os.environ.get("PORT", 5000)))