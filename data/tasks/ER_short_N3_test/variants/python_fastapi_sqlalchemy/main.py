import os

from fastapi import FastAPI
from fastapi.responses import JSONResponse
from sqlalchemy import Column, Integer, Text, Float, create_engine
from sqlalchemy.orm import declarative_base, sessionmaker, Session

DATABASE = os.environ.get("DB_PATH", "app.db")
DATABASE_URL = f"sqlite:///{DATABASE}"

engine = create_engine(
    DATABASE_URL,
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


@app.post("/api/products/{p_productkey}/archive")
def archive_product(p_productkey: int):
    db: Session = SessionLocal()
    try:
        existing = (
            db.query(Product.p_productkey)
            .filter(Product.p_productkey == p_productkey)
            .first()
        )
        if existing is None:
            return JSONResponse({"error": "Product not found"}, status_code=404)

        db.query(Product).filter(Product.p_productkey == p_productkey).update(
            {"p_status": "archived"}, synchronize_session=False
        )
        db.commit()
        return JSONResponse(
            {"p_productkey": p_productkey, "p_status": "archived"}, status_code=200
        )
    finally:
        db.close()


@app.get("/api/products/archived")
def list_archived_products():
    db: Session = SessionLocal()
    try:
        rows = (
            db.query(
                Product.p_productkey,
                Product.p_name,
                Product.p_retailprice,
            )
            .filter(Product.p_status == "archived")
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