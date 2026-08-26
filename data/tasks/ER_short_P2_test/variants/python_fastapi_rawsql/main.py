import os
import sqlite3
from fastapi import FastAPI
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


@app.post("/api/products/{p_productkey}/discontinue")
def discontinue_product(p_productkey: int):
    db = get_db()
    try:
        existing = db.execute(
            "SELECT p_productkey FROM product WHERE p_productkey = ?", (p_productkey,)
        ).fetchone()
        if existing is None:
            return JSONResponse({"error": "Product not found"}, status_code=404)
        db.execute(
            "UPDATE product SET p_status = 'discontinued' WHERE p_productkey = ?",
            (p_productkey,),
        )
        db.commit()
        return JSONResponse({"p_productkey": p_productkey, "p_status": "discontinued"}, status_code=200)
    finally:
        db.close()


@app.get("/api/products/{p_productkey}/availability")
def get_product_availability(p_productkey: int):
    """
    Get availability for purchasable product
    """
    db = get_db()
    try:
        row = db.execute(
            "SELECT p.p_productkey, p.p_name, "
            "       COALESCE(SUM(i.i_quantity), 0) AS total_stock "
            "FROM product p LEFT JOIN inventory i ON p.p_productkey = i.i_productkey "
            "WHERE p.p_productkey = ? "
            "GROUP BY p.p_productkey",
            (p_productkey,),
        ).fetchone()
        if row is None:
            return JSONResponse({"error": "Product not found"}, status_code=404)
        return JSONResponse(dict(row), status_code=200)
    finally:
        db.close()


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=int(os.environ.get("PORT", 5000)))