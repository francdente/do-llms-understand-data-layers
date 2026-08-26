import os
from fastapi import FastAPI
from fastapi.responses import JSONResponse
from sqlalchemy import create_engine, Column, Integer, Text, Float, ForeignKey, func
from sqlalchemy.orm import declarative_base, sessionmaker, Session

app = FastAPI()
DATABASE = os.environ.get("DB_PATH", "app.db")

engine = create_engine(f"sqlite:///{DATABASE}", connect_args={"check_same_thread": False})
SessionLocal = sessionmaker(bind=engine, autoflush=False, autocommit=False)

Base = declarative_base()


class Product(Base):
    __tablename__ = "product"

    p_productkey = Column(Integer, primary_key=True)
    p_name = Column(Text)
    p_status = Column(Text)
    p_retailprice = Column(Float)


class Inventory(Base):
    __tablename__ = "inventory"

    i_productkey = Column(Integer, ForeignKey("product.p_productkey"), primary_key=True)
    i_warehouse = Column(Text, primary_key=True)
    i_quantity = Column(Integer)


@app.post("/api/products/{p_productkey}/discontinue")
def discontinue_product(p_productkey: int):
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
            {Product.p_status: "discontinued"}, synchronize_session=False
        )
        db.commit()
        return JSONResponse(
            {"p_productkey": p_productkey, "p_status": "discontinued"}, status_code=200
        )
    finally:
        db.close()


@app.get("/api/products/{p_productkey}/availability")
def get_product_availability(p_productkey: int):
    """
    Get availability for purchasable product
    """
    db: Session = SessionLocal()
    try:
        row = (
            db.query(
                Product.p_productkey.label("p_productkey"),
                Product.p_name.label("p_name"),
                func.coalesce(func.sum(Inventory.i_quantity), 0).label("total_stock"),
            )
            .select_from(Product)
            .outerjoin(Inventory, Product.p_productkey == Inventory.i_productkey)
            .filter(Product.p_productkey == p_productkey)
            .group_by(Product.p_productkey)
            .first()
        )
        if row is None:
            return JSONResponse({"error": "Product not found"}, status_code=404)
        return JSONResponse(
            {
                "p_productkey": row.p_productkey,
                "p_name": row.p_name,
                "total_stock": row.total_stock,
            },
            status_code=200,
        )
    finally:
        db.close()


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=int(os.environ.get("PORT", 5000)))