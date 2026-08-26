import os
from fastapi import FastAPI, Depends
from fastapi.responses import JSONResponse
from sqlalchemy import create_engine, Column, Integer, Text, Float, ForeignKey
from sqlalchemy.orm import declarative_base, relationship, sessionmaker, Session

DATABASE = os.environ.get("DB_PATH", "app.db")

engine = create_engine(f"sqlite:///{DATABASE}", connect_args={"check_same_thread": False})
SessionLocal = sessionmaker(bind=engine, autoflush=False, autocommit=False)

Base = declarative_base()


class Supplier(Base):
    __tablename__ = "supplier"

    s_suppkey = Column(Integer, primary_key=True)
    s_name = Column(Text)
    s_status = Column(Text)

    products = relationship("Product", back_populates="supplier")


class Product(Base):
    __tablename__ = "product"

    p_productkey = Column(Integer, primary_key=True)
    p_name = Column(Text)
    p_suppkey = Column(Integer, ForeignKey("supplier.s_suppkey"))
    p_retailprice = Column(Float)

    supplier = relationship("Supplier", back_populates="products")


app = FastAPI()


def get_db():
    db = SessionLocal()
    try:
        yield db
    finally:
        db.close()


@app.post("/api/suppliers/{s_suppkey}/suspend")
def suspend_supplier(s_suppkey: int, db: Session = Depends(get_db)):
    existing = db.query(Supplier.s_suppkey).filter(Supplier.s_suppkey == s_suppkey).first()
    if existing is None:
        return JSONResponse({"error": "Supplier not found"}, status_code=404)
    db.query(Supplier).filter(Supplier.s_suppkey == s_suppkey).update(
        {"s_status": "suspended"}, synchronize_session=False
    )
    db.commit()
    return JSONResponse({"s_suppkey": s_suppkey, "s_status": "suspended"}, status_code=200)


@app.get("/api/products")
def list_products(db: Session = Depends(get_db)):
    """
    List purchasable products
    """
    rows = (
        db.query(
            Product.p_productkey,
            Product.p_name,
            Product.p_retailprice,
            Supplier.s_name.label("supplier_name"),
        )
        .join(Supplier, Product.p_suppkey == Supplier.s_suppkey)
        .order_by(Product.p_productkey)
        .all()
    )
    return JSONResponse(
        [
            {
                "p_productkey": row.p_productkey,
                "p_name": row.p_name,
                "p_retailprice": row.p_retailprice,
                "supplier_name": row.supplier_name,
            }
            for row in rows
        ],
        status_code=200,
    )


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=int(os.environ.get("PORT", 5000)))