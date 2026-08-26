import os
import sqlite3
from fastapi import FastAPI, Response
from fastapi.responses import JSONResponse
from contextlib import asynccontextmanager

DATABASE = os.environ.get("DB_PATH", "app.db")


def get_db():
    db = sqlite3.connect(DATABASE)
    db.row_factory = sqlite3.Row
    return db


@asynccontextmanager
async def lifespan(app: FastAPI):
    yield


app = FastAPI(lifespan=lifespan)


@app.delete("/api/customers/{c_custkey}")
def delete_customer(c_custkey: int):
    db = get_db()
    try:
        existing = db.execute(
            "SELECT c_custkey FROM customer WHERE c_custkey = ?", (c_custkey,)
        ).fetchone()
        if existing is None:
            return JSONResponse({"error": "Customer not found"}, status_code=404)
        db.execute("DELETE FROM orders WHERE o_custkey = ?", (c_custkey,))
        db.execute("DELETE FROM customer WHERE c_custkey = ?", (c_custkey,))
        db.commit()
        return Response(status_code=204)
    finally:
        db.close()


@app.get("/api/orders")
def list_orders():
    """
    Return orders tied to active customers
    """
    db = get_db()
    try:
        rows = db.execute(
            "SELECT o_orderkey, o_custkey, o_totalprice, o_orderdate, o_comment "
            "FROM orders WHERE o_custkey IN (SELECT c_custkey FROM customer) "
            "ORDER BY o_orderdate"
        ).fetchall()
        return JSONResponse([dict(r) for r in rows], status_code=200)
    finally:
        db.close()


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=int(os.environ.get("PORT", 5000)))