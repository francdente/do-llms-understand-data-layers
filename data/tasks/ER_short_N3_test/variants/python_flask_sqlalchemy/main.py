import os
from flask import Flask, jsonify
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
def shutdown_session(exception=None):
    SessionLocal.remove()


@app.route("/api/products/<int:p_productkey>/archive", methods=["POST"])
def archive_product(p_productkey):
    session = SessionLocal()
    existing = (
        session.query(Product.p_productkey)
        .filter(Product.p_productkey == p_productkey)
        .first()
    )
    if existing is None:
        return jsonify({"error": "Product not found"}), 404

    session.query(Product).filter(Product.p_productkey == p_productkey).update(
        {"p_status": "archived"}, synchronize_session=False
    )
    session.commit()
    return jsonify({"p_productkey": p_productkey, "p_status": "archived"}), 200


@app.route("/api/products/archived", methods=["GET"])
def list_archived_products():
    session = SessionLocal()
    rows = (
        session.query(
            Product.p_productkey,
            Product.p_name,
            Product.p_retailprice,
        )
        .filter(Product.p_status == "archived")
        .order_by(Product.p_productkey)
        .all()
    )
    return (
        jsonify(
            [
                {
                    "p_productkey": row.p_productkey,
                    "p_name": row.p_name,
                    "p_retailprice": row.p_retailprice,
                }
                for row in rows
            ]
        ),
        200,
    )


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=int(os.environ.get("PORT", 5000)))