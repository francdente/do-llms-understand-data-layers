import os
import sqlite3
from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse
from contextlib import asynccontextmanager

DATABASE = os.environ.get("DB_PATH", "app.db")


@asynccontextmanager
async def lifespan(app: FastAPI):
    yield


app = FastAPI(lifespan=lifespan)


def get_db(request: Request):
    db = getattr(request.state, "_database", None)
    if db is None:
        db = sqlite3.connect(DATABASE)
        db.row_factory = sqlite3.Row
        request.state._database = db
    return db


@app.middleware("http")
async def db_session_middleware(request: Request, call_next):
    response = None
    try:
        response = await call_next(request)
        return response
    finally:
        db = getattr(request.state, "_database", None)
        if db is not None:
            db.close()


@app.get("/api/orders/{o_orderkey}")
async def get_order(o_orderkey: int, request: Request):
    db = get_db(request)
    row = db.execute(
        "SELECT o_orderkey, o_orderstatus, o_totalprice, o_orderdate, o_comment "
        "FROM orders WHERE o_orderkey = ?",
        (o_orderkey,),
    ).fetchone()
    if row is None:
        return JSONResponse({"error": "Order not found"}, status_code=404)
    return JSONResponse(dict(row), status_code=200)


@app.put("/api/orders/{o_orderkey}/lineitems/{l_linenumber}")
async def update_lineitem(o_orderkey: int, l_linenumber: int, request: Request):
    payload = await request.json()
    if payload is None:
        payload = {}
    db = get_db(request)
    db.execute("BEGIN")
    existing = db.execute(
        "SELECT l_orderkey FROM lineitem WHERE l_orderkey = ? AND l_linenumber = ?",
        (o_orderkey, l_linenumber),
    ).fetchone()
    if existing is None:
        db.execute("ROLLBACK")
        return JSONResponse({"error": "Line item not found"}, status_code=404)
    db.execute(
        "UPDATE lineitem SET l_extendedprice = ?, l_discount = ?, l_tax = ? "
        "WHERE l_orderkey = ? AND l_linenumber = ?",
        (payload["l_extendedprice"], payload["l_discount"], payload["l_tax"],
         o_orderkey, l_linenumber),
    )
    db.execute(
        "UPDATE orders SET o_totalprice = "
        "(SELECT COALESCE(SUM(l_extendedprice * (1 - l_discount) * (1 + l_tax)), 0) "
        " FROM lineitem WHERE l_orderkey = ?) "
        "WHERE o_orderkey = ?",
        (o_orderkey, o_orderkey),
    )
    row = db.execute(
        "SELECT l_orderkey, l_linenumber, l_extendedprice, l_discount, l_tax "
        "FROM lineitem WHERE l_orderkey = ? AND l_linenumber = ?",
        (o_orderkey, l_linenumber),
    ).fetchone()
    db.commit()
    return JSONResponse(dict(row), status_code=200)


@app.post("/api/orders")
async def create_order(request: Request):
    payload = await request.json()
    if payload is None:
        payload = {}
    db = get_db(request)
    db.execute("BEGIN")
    cur = db.execute(
        "INSERT INTO orders (o_orderstatus, o_totalprice, o_orderdate, o_comment) "
        "VALUES (?, 0, ?, ?)",
        (payload.get("o_orderstatus", "O"), payload["o_orderdate"],
         payload.get("o_comment", "")),
    )
    o_orderkey = cur.lastrowid
    for item in payload["lineitems"]:
        db.execute(
            "INSERT INTO lineitem (l_orderkey, l_linenumber, l_extendedprice, "
            "l_discount, l_tax, l_shipdate, l_comment) "
            "VALUES (?, ?, ?, ?, ?, ?, ?)",
            (o_orderkey, item["l_linenumber"], item["l_extendedprice"],
             item["l_discount"], item["l_tax"],
             item.get("l_shipdate"), item.get("l_comment", "")),
        )
    total = db.execute(
        "SELECT COALESCE(SUM(l_extendedprice * (1 - l_discount) * (1 + l_tax)), 0) AS total "
        "FROM lineitem WHERE l_orderkey = ?",
        (o_orderkey,),
    ).fetchone()["total"]
    db.execute(
        "UPDATE orders SET o_totalprice = ? WHERE o_orderkey = ?",
        (total, o_orderkey),
    )
    db.commit()
    return JSONResponse({"o_orderkey": o_orderkey, "o_totalprice": total}, status_code=201)


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=int(os.environ.get("PORT", 5000)))