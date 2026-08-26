import os
import sqlite3
from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse
import uvicorn

app = FastAPI()
DATABASE = os.environ.get("DB_PATH", "app.db")


def get_db():
    db = sqlite3.connect(DATABASE)
    db.row_factory = sqlite3.Row
    return db


@app.put("/api/products/{p_productkey}/price")
async def update_product_price(p_productkey: int, request: Request):
    try:
        payload = await request.json()
        if payload is None:
            payload = {}
    except Exception:
        payload = {}

    db = get_db()
    try:
        existing = db.execute(
            "SELECT p_productkey FROM product WHERE p_productkey = ?", (p_productkey,)
        ).fetchone()
        if existing is None:
            return JSONResponse({"error": "Product not found"}, status_code=404)

        db.execute(
            "UPDATE product SET p_retailprice = ? WHERE p_productkey = ?",
            (payload["p_retailprice"], p_productkey),
        )
        db.commit()
        row = db.execute(
            "SELECT p_productkey, p_name, p_status, p_retailprice "
            "FROM product WHERE p_productkey = ?",
            (p_productkey,),
        ).fetchone()
        return JSONResponse(dict(row), status_code=200)
    finally:
        db.close()


@app.get("/api/products/{p_productkey}")
async def get_product(p_productkey: int):
    db = get_db()
    try:
        row = db.execute(
            "SELECT p_productkey, p_name, p_status, p_retailprice "
            "FROM product WHERE p_productkey = ?",
            (p_productkey,),
        ).fetchone()
        if row is None:
            return JSONResponse({"error": "Product not found"}, status_code=404)
        return JSONResponse(dict(row), status_code=200)
    finally:
        db.close()


if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=int(os.environ.get("PORT", 5000)))