import os
from typing import Generator

from fastapi import FastAPI
from fastapi.responses import JSONResponse
from sqlalchemy import Column, Float, Integer, String, create_engine
from sqlalchemy.orm import declarative_base, sessionmaker, Session

app = FastAPI()
DATABASE = os.environ.get("DB_PATH", "app.db")

engine = create_engine(f"sqlite:///{DATABASE}", future=True)
SessionLocal = sessionmaker(bind=engine, autoflush=False, autocommit=False, future=True)
Base = declarative_base()


class Product(Base):
    __tablename__ = "product"

    p_productkey = Column(Integer, primary_key=True)
    p_name = Column(String)
    p_status = Column(String)
    p_retailprice = Column(Float)


def get_db() -> Generator[Session, None, None]:
    db = SessionLocal()
    try:
        yield db
    finally:
        db.close()


@app.post("/api/products/{p_productkey}/discontinue")
def discontinue_product(p_productkey: int):
    db = SessionLocal()
    try:
        existing = (
            db.query(Product.p_productkey)
            .filter(Product.p_productkey == p_productkey)
            .first()
        )
        if existing is None:
            return JSONResponse({"error": "Product not found"}, status_code=404)
        product = db.get(Product, p_productkey)
        product.p_status = "discontinued"
        db.commit()
        return JSONResponse(
            {"p_productkey": p_productkey, "p_status": "discontinued"}, status_code=200
        )
    finally:
        db.close()


@app.get("/api/products")
def list_products():
    """
    List purchasable products
    """
    db = SessionLocal()
    try:
        rows = (
            db.query(
                Product.p_productkey,
                Product.p_name,
                Product.p_retailprice,
            )
            .filter(Product.p_status == "active")
            .order_by(Product.p_productkey)
            .all()
        )
        return JSONResponse(
            [
                {
                    "p_productkey": row.p_productkey,
                    "p_name": row.p_name,
                    "p_retailprice": row.p_retailprice,
                }
                for row in rows
            ],
            status_code=200,
        )
    finally:
        db.close()


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=int(os.environ.get("PORT", 5000)))