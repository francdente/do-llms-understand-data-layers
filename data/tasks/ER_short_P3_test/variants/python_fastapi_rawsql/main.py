import os
import sqlite3
from fastapi import FastAPI
from fastapi.responses import JSONResponse
from starlette.requests import Request

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


@app.post("/api/suppliers/{s_suppkey}/suspend")
def suspend_supplier(s_suppkey: int, request: Request):
    db = get_db(request)
    existing = db.execute(
        "SELECT s_suppkey FROM supplier WHERE s_suppkey = ?", (s_suppkey,)
    ).fetchone()
    if existing is None:
        return JSONResponse({"error": "Supplier not found"}, status_code=404)
    db.execute(
        "UPDATE supplier SET s_status = 'suspended' WHERE s_suppkey = ?",
        (s_suppkey,),
    )
    db.commit()
    return JSONResponse({"s_suppkey": s_suppkey, "s_status": "suspended"}, status_code=200)


@app.get("/api/products")
def list_products(request: Request):
    """
    List purchasable products
    """
    db = get_db(request)
    rows = db.execute(
        "SELECT p.p_productkey, p.p_name, p.p_retailprice, s.s_name AS supplier_name "
        "FROM product p JOIN supplier s ON p.p_suppkey = s.s_suppkey "
        "ORDER BY p.p_productkey"
    ).fetchall()
    return JSONResponse([dict(r) for r in rows], status_code=200)


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=int(os.environ.get("PORT", 5000)))