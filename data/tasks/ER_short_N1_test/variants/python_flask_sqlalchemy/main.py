import os
from flask import Flask, jsonify, g
from sqlalchemy import create_engine, Column, Integer, Text, Float
from sqlalchemy.orm import declarative_base, sessionmaker

app = Flask(__name__)
DATABASE = os.environ.get("DB_PATH", "app.db")

engine = create_engine(f"sqlite:///{DATABASE}")
SessionLocal = sessionmaker(bind=engine)

Base = declarative_base()


class Product(Base):
    __tablename__ = "product"

    p_productkey = Column(Integer, primary_key=True)
    p_name = Column(Text)
    p_status = Column(Text)
    p_retailprice = Column(Float)


def get_db():
    db = getattr(g, "_database", None)
    if db is None:
        db = g._database = SessionLocal()
    return db


@app.teardown_appcontext
def close_db(exception):
    db = getattr(g, "_database", None)
    if db is not None:
        db.close()


@app.route("/api/products/<int:p_productkey>/discontinue", methods=["POST"])
def discontinue_product(p_productkey):
    db = get_db()
    existing = db.query(Product.p_productkey).filter(Product.p_productkey == p_productkey).first()
    if existing is None:
        return jsonify({"error": "Product not found"}), 404
    db.query(Product).filter(Product.p_productkey == p_productkey).update(
        {Product.p_status: "discontinued"}, synchronize_session=False
    )
    db.commit()
    return jsonify({"p_productkey": p_productkey, "p_status": "discontinued"}), 200


@app.route("/api/products", methods=["GET"])
def list_products():
    """
    List purchasable products
    """
    db = get_db()
    rows = (
        db.query(Product.p_productkey, Product.p_name, Product.p_retailprice)
        .filter(Product.p_status == "active")
        .order_by(Product.p_productkey)
        .all()
    )
    return jsonify(
        [
            {
                "p_productkey": row.p_productkey,
                "p_name": row.p_name,
                "p_retailprice": row.p_retailprice,
            }
            for row in rows
        ]
    ), 200


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=int(os.environ.get("PORT", 5000)))