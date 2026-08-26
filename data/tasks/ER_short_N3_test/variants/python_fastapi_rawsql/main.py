import os
import sqlite3
from fastapi import FastAPI
from fastapi.responses import JSONResponse
from typing import Generator

app = FastAPI()
DATABASE = os.environ.get("DB_PATH", "app.db")


def get_db() -> sqlite3.Connection:
    db = sqlite3.connect(DATABASE)
    db.row_factory = sqlite3.Row
    return db


@app.post("/api/products/{p_productkey}/archive")
def archive_product(p_productkey: int):
    db = get_db()
    try:
        existing = db.execute(
            "SELECT p_productkey FROM product WHERE p_productkey = ?", (p_productkey,)
        ).fetchone()
        if existing is None:
            return JSONResponse({"error": "Product not found"}, status_code=404)
        db.execute(
            "UPDATE product SET p_status = 'archived' WHERE p_productkey = ?",
            (p_productkey,),
        )
        db.commit()
        return JSONResponse(
            {"p_productkey": p_productkey, "p_status": "archived"}, status_code=200
        )
    finally:
        db.close()


@app.get("/api/products/archived")
def list_archived_products():
    db = get_db()
    try:
        rows = db.execute(
            "SELECT p_productkey, p_name, p_retailprice "
            "FROM product WHERE p_status = 'archived' "
            "ORDER BY p_productkey"
        ).fetchall()
        return JSONResponse([dict(r) for r in rows], status_code=200)
    finally:
        db.close()


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=int(os.environ.get("PORT", 5000)))