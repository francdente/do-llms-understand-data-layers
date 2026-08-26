import os
import sqlite3
from datetime import date
from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse

app = FastAPI()
DATABASE = os.environ.get("DB_PATH", "app.db")


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


@app.post("/api/orders/{o_orderkey}/lineitems/{l_linenumber}/ship")
def ship_lineitem(o_orderkey: int, l_linenumber: int, request: Request):
    db = get_db(request)
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
    latest = db.execute(
        "SELECT MAX(l_shipdate) AS latest FROM lineitem WHERE l_orderkey = ?",
        (o_orderkey,),
    ).fetchone()["latest"]
    db.execute(
        "UPDATE orders SET o_latest_shipdate = ? WHERE o_orderkey = ?",
        (latest, o_orderkey),
    )
    db.commit()
    return JSONResponse(
        {"l_orderkey": o_orderkey, "l_linenumber": l_linenumber, "l_shipdate": today},
        status_code=200,
    )


@app.get("/api/orders/{o_orderkey}")
def get_order(o_orderkey: int, request: Request):
    db = get_db(request)
    row = db.execute(
        "SELECT o_orderkey, o_totalprice, o_orderdate, o_latest_shipdate, o_comment "
        "FROM orders WHERE o_orderkey = ?",
        (o_orderkey,),
    ).fetchone()
    if row is None:
        return JSONResponse({"error": "Order not found"}, status_code=404)
    return JSONResponse(dict(row), status_code=200)


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=int(os.environ.get("PORT", 5000)))