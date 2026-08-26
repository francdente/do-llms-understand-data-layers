import os
from flask import Flask, jsonify
from sqlalchemy import create_engine, Column, Integer, Text, Float, ForeignKey
from sqlalchemy.orm import declarative_base, relationship, sessionmaker, scoped_session

app = Flask(__name__)
DATABASE = os.environ.get("DB_PATH", "app.db")

engine = create_engine(f"sqlite:///{DATABASE}")
SessionLocal = scoped_session(sessionmaker(bind=engine))

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


@app.teardown_appcontext
def close_db(exception):
    SessionLocal.remove()


@app.route("/api/suppliers/<int:s_suppkey>/suspend", methods=["POST"])
def suspend_supplier(s_suppkey):
    db = SessionLocal()
    existing = db.query(Supplier.s_suppkey).filter(Supplier.s_suppkey == s_suppkey).first()
    if existing is None:
        return jsonify({"error": "Supplier not found"}), 404
    db.query(Supplier).filter(Supplier.s_suppkey == s_suppkey).update(
        {"s_status": "suspended"}, synchronize_session=False
    )
    db.commit()
    return jsonify({"s_suppkey": s_suppkey, "s_status": "suspended"}), 200


@app.route("/api/products", methods=["GET"])
def list_products():
    """
    List purchasable products
    """
    db = SessionLocal()
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
    return jsonify(
        [
            {
                "p_productkey": row.p_productkey,
                "p_name": row.p_name,
                "p_retailprice": row.p_retailprice,
                "supplier_name": row.supplier_name,
            }
            for row in rows
        ]
    ), 200


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=int(os.environ.get("PORT", 5000)))