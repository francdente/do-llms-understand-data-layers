import os
import sqlite3
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


@app.post("/api/products/{p_productkey}/discontinue")
async def discontinue_product(p_productkey: int, request: Request):
    db = get_db(request)
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


@app.get("/api/products/{p_productkey}")
async def get_product(p_productkey: int, request: Request):
    """
    Return purchasable product
    """
    db = get_db(request)
    row = db.execute(
        "SELECT p_productkey, p_name, p_status, p_retailprice "
        "FROM product WHERE p_productkey = ?",
        (p_productkey,),
    ).fetchone()
    if row is None:
        return JSONResponse({"error": "Product not found"}, status_code=404)
    return JSONResponse(dict(row), status_code=200)


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=int(os.environ.get("PORT", 5000)))