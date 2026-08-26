import os

from fastapi import FastAPI, Depends, HTTPException, Request
from fastapi.responses import JSONResponse
from sqlalchemy import create_engine, Column, Integer, Text, Float
from sqlalchemy.orm import declarative_base, sessionmaker, Session

DATABASE = os.environ.get("DB_PATH", "app.db")

engine = create_engine(
    f"sqlite:///{DATABASE}",
    connect_args={"check_same_thread": False},
)
SessionLocal = sessionmaker(bind=engine, autoflush=False, autocommit=False)

Base = declarative_base()


class Product(Base):
    __tablename__ = "product"

    p_productkey = Column(Integer, primary_key=True)
    p_name = Column(Text)
    p_status = Column(Text)
    p_retailprice = Column(Float)


app = FastAPI()


def get_db():
    db = SessionLocal()
    try:
        yield db
    finally:
        db.close()


def product_to_dict(product: Product):
    return {
        "p_productkey": product.p_productkey,
        "p_name": product.p_name,
        "p_status": product.p_status,
        "p_retailprice": product.p_retailprice,
    }


@app.exception_handler(HTTPException)
async def http_exception_handler(request: Request, exc: HTTPException):
    if isinstance(exc.detail, dict):
        return JSONResponse(status_code=exc.status_code, content=exc.detail)
    return JSONResponse(status_code=exc.status_code, content={"detail": exc.detail})


@app.put("/api/products/{p_productkey}/price")
def update_product_price(p_productkey: int, request: Request, db: Session = Depends(get_db)):
    try:
        payload = request.json() if False else None
    except Exception:
        payload = None
    # request.json() is async in FastAPI/Starlette; use body parsing manually
    # while preserving Flask's silent=True behavior.
    import json
    from starlette.requests import Request as StarletteRequest

    if isinstance(request, StarletteRequest):
        body_bytes = None
        try:
            body_bytes = request._body if hasattr(request, "_body") else None
        except Exception:
            body_bytes = None
        if body_bytes is None:
            import anyio
            body_bytes = anyio.run(request.body)
        try:
            parsed = json.loads(body_bytes.decode("utf-8")) if body_bytes else None
        except Exception:
            parsed = None
    else:
        parsed = None

    payload = parsed or {}

    existing = db.query(Product.p_productkey).filter(Product.p_productkey == p_productkey).first()
    if existing is None:
        raise HTTPException(status_code=404, detail={"error": "Product not found"})

    db.query(Product).filter(Product.p_productkey == p_productkey).update(
        {"p_retailprice": payload["p_retailprice"]},
        synchronize_session=False,
    )
    db.commit()

    row = (
        db.query(
            Product.p_productkey,
            Product.p_name,
            Product.p_status,
            Product.p_retailprice,
        )
        .filter(Product.p_productkey == p_productkey)
        .first()
    )

    return {
        "p_productkey": row.p_productkey,
        "p_name": row.p_name,
        "p_status": row.p_status,
        "p_retailprice": row.p_retailprice,
    }


@app.get("/api/products/{p_productkey}")
def get_product(p_productkey: int, db: Session = Depends(get_db)):
    row = (
        db.query(
            Product.p_productkey,
            Product.p_name,
            Product.p_status,
            Product.p_retailprice,
        )
        .filter(Product.p_productkey == p_productkey)
        .first()
    )
    if row is None:
        raise HTTPException(status_code=404, detail={"error": "Product not found"})
    return {
        "p_productkey": row.p_productkey,
        "p_name": row.p_name,
        "p_status": row.p_status,
        "p_retailprice": row.p_retailprice,
    }


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=int(os.environ.get("PORT", 5000)))