import os
from flask import Flask, request, jsonify
from sqlalchemy import create_engine, Column, Integer, Text, Float
from sqlalchemy.orm import declarative_base, sessionmaker, scoped_session

app = Flask(__name__)
DATABASE = os.environ.get("DB_PATH", "app.db")

engine = create_engine(f"sqlite:///{DATABASE}")
SessionLocal = scoped_session(sessionmaker(bind=engine))

Base = declarative_base()


class Product(Base):
    __tablename__ = "product"

    p_productkey = Column(Integer, primary_key=True)
    p_name = Column(Text)
    p_status = Column(Text)
    p_retailprice = Column(Float)


@app.teardown_appcontext
def close_db(exception):
    SessionLocal.remove()


@app.route("/api/products/<int:p_productkey>/price", methods=["PUT"])
def update_product_price(p_productkey):
    payload = request.get_json(silent=True) or {}
    db = SessionLocal()

    existing = (
        db.query(Product.p_productkey)
        .filter(Product.p_productkey == p_productkey)
        .first()
    )
    if existing is None:
        return jsonify({"error": "Product not found"}), 404

    db.query(Product).filter(Product.p_productkey == p_productkey).update(
        {Product.p_retailprice: payload["p_retailprice"]},
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

    return jsonify({
        "p_productkey": row.p_productkey,
        "p_name": row.p_name,
        "p_status": row.p_status,
        "p_retailprice": row.p_retailprice,
    }), 200


@app.route("/api/products/<int:p_productkey>", methods=["GET"])
def get_product(p_productkey):
    db = SessionLocal()
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
        return jsonify({"error": "Product not found"}), 404

    return jsonify({
        "p_productkey": row.p_productkey,
        "p_name": row.p_name,
        "p_status": row.p_status,
        "p_retailprice": row.p_retailprice,
    }), 200


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=int(os.environ.get("PORT", 5000)))