import os
from typing import Generator

from fastapi import FastAPI, Depends
from fastapi.responses import JSONResponse
from sqlalchemy import create_engine, Column, Integer, Text, Float
from sqlalchemy.orm import declarative_base, sessionmaker, Session

DATABASE = os.environ.get("DB_PATH", "app.db")
DATABASE_URL = f"sqlite:///{DATABASE}"

engine = create_engine(DATABASE_URL, connect_args={"check_same_thread": False})
SessionLocal = sessionmaker(bind=engine, autoflush=False, autocommit=False)

Base = declarative_base()


class Product(Base):
    __tablename__ = "product"

    p_productkey = Column(Integer, primary_key=True)
    p_name = Column(Text)
    p_status = Column(Text)
    p_retailprice = Column(Float)


app = FastAPI()


def get_db() -> Generator[Session, None, None]:
    db = SessionLocal()
    try:
        yield db
    finally:
        db.close()


@app.post("/api/products/{p_productkey}/discontinue")
def discontinue_product(p_productkey: int, db: Session = Depends(get_db)):
    existing = (
        db.query(Product.p_productkey)
        .filter(Product.p_productkey == p_productkey)
        .first()
    )
    if existing is None:
        return JSONResponse({"error": "Product not found"}, status_code=404)
    db.query(Product).filter(Product.p_productkey == p_productkey).update(
        {"p_status": "discontinued"}, synchronize_session=False
    )
    db.commit()
    return JSONResponse(
        {"p_productkey": p_productkey, "p_status": "discontinued"}, status_code=200
    )


@app.get("/api/products/{p_productkey}")
def get_product(p_productkey: int, db: Session = Depends(get_db)):
    """
    Return purchasable product
    """
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
        return JSONResponse({"error": "Product not found"}, status_code=404)
    return JSONResponse(
        {
            "p_productkey": row.p_productkey,
            "p_name": row.p_name,
            "p_status": row.p_status,
            "p_retailprice": row.p_retailprice,
        },
        status_code=200,
    )


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=int(os.environ.get("PORT", 5000)))