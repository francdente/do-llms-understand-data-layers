import os
import sqlite3
from datetime import date
from fastapi import FastAPI
from fastapi.responses import JSONResponse
from contextlib import asynccontextmanager

DATABASE = os.environ.get("DB_PATH", "app.db")


@asynccontextmanager
async def lifespan(app: FastAPI):
    yield


app = FastAPI(lifespan=lifespan)


def get_db():
    db = sqlite3.connect(DATABASE)
    db.row_factory = sqlite3.Row
    return db


@app.post("/api/orders/{o_orderkey}/lineitems/{l_linenumber}/ship")
def ship_lineitem(o_orderkey: int, l_linenumber: int):
    db = get_db()
    try:
        existing = db.execute(
            "SELECT l_orderkey FROM lineitem WHERE l_orderkey = ? AND l_linenumber = ?",
            (o_orderkey, l_linenumber),
        ).fetchone()
        if existing is None:
            return JSONResponse({"error": "Line item not found"}, status_code=404)
        today = date.today().isoformat()
        db.execute(
            "UPDATE lineitem SET l_shipdate = ? "
            "WHERE l_orderkey = ? AND l_linenumber = ?",
            (today, o_orderkey, l_linenumber),
        )
        db.commit()
        return JSONResponse(
            {"l_orderkey": o_orderkey, "l_linenumber": l_linenumber, "l_shipdate": today},
            status_code=200,
        )
    finally:
        db.close()


@app.get("/api/orders/{o_orderkey}")
def get_order(o_orderkey: int):
    db = get_db()
    try:
        row = db.execute(
            "SELECT o_orderkey, o_totalprice, o_orderdate, o_latest_shipdate, o_comment "
            "FROM orders WHERE o_orderkey = ?",
            (o_orderkey,),
        ).fetchone()
        if row is None:
            return JSONResponse({"error": "Order not found"}, status_code=404)
        return JSONResponse(dict(row), status_code=200)
    finally:
        db.close()


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=int(os.environ.get("PORT", 5000)))