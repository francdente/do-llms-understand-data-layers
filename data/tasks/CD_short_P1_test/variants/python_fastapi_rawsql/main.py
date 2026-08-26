import os
import sqlite3
from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse, Response

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


@app.delete("/api/customers/{c_custkey}")
def delete_customer(c_custkey: int, request: Request):
    db = get_db(request)
    existing = db.execute(
        "SELECT c_custkey FROM customer WHERE c_custkey = ?", (c_custkey,)
    ).fetchone()
    if existing is None:
        return JSONResponse({"error": "Customer not found"}, status_code=404)
    db.execute("DELETE FROM orders WHERE o_custkey = ?", (c_custkey,))
    db.execute("DELETE FROM customer WHERE c_custkey = ?", (c_custkey,))
    db.commit()
    return Response(status_code=204)


@app.get("/api/orders/summary")
def orders_summary(request: Request):
    """
    Return the summary of all existed historical orders
    """
    db = get_db(request)
    row = db.execute(
        "SELECT COUNT(*) AS total_orders, "
        "       COALESCE(SUM(o_totalprice), 0) AS total_revenue "
        "FROM orders"
    ).fetchone()
    return JSONResponse(dict(row), status_code=200)


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=int(os.environ.get("PORT", 5000)))