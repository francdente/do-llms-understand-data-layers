import os
from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.responses import JSONResponse, Response
from sqlalchemy import Column, ForeignKey, Integer, Real, Text, create_engine, event, func, select
from sqlalchemy.exc import IntegrityError
from sqlalchemy.orm import declarative_base, sessionmaker

DATABASE = os.environ.get("DB_PATH", "app.db")
DATABASE_URL = f"sqlite:///{DATABASE}"

engine = create_engine(DATABASE_URL, future=True)


@event.listens_for(engine, "connect")
def set_sqlite_pragma(dbapi_connection, connection_record):
    cursor = dbapi_connection.cursor()
    cursor.execute("PRAGMA foreign_keys = ON;")
    cursor.close()


SessionLocal = sessionmaker(bind=engine, autoflush=False, autocommit=False, future=True)
Base = declarative_base()


class Customer(Base):
    __tablename__ = "customer"

    c_custkey = Column(Integer, primary_key=True)
    c_name = Column(Text)
    c_email = Column(Text)


class Order(Base):
    __tablename__ = "orders"

    o_orderkey = Column(Integer, primary_key=True)
    o_custkey = Column(Integer, ForeignKey("customer.c_custkey"))
    o_totalprice = Column(Real)
    o_orderdate = Column(Text)
    o_comment = Column(Text)


@asynccontextmanager
async def lifespan(app: FastAPI):
    yield
    engine.dispose()


app = FastAPI(lifespan=lifespan)


@app.delete("/api/customers/{c_custkey}")
def delete_customer(c_custkey: int):
    with SessionLocal() as db:
        existing = db.execute(
            select(Customer.c_custkey).where(Customer.c_custkey == c_custkey)
        ).first()
        if existing is None:
            return JSONResponse({"error": "Customer not found"}, status_code=404)
        try:
            db.query(Customer).filter(Customer.c_custkey == c_custkey).delete(synchronize_session=False)
            db.commit()
        except IntegrityError:
            db.rollback()
            return JSONResponse({"error": "Cannot delete customer with existing orders"}, status_code=409)
        return Response(status_code=204)


@app.get("/api/dashboard")
def dashboard():
    """
    Return dashboard based on all existed orders
    """
    with SessionLocal() as db:
        row = db.execute(
            select(
                func.count().label("total_orders"),
                func.coalesce(func.sum(Order.o_totalprice), 0).label("total_revenue"),
            ).select_from(Order)
        ).one()
        return JSONResponse(
            {
                "total_orders": row.total_orders,
                "total_revenue": row.total_revenue,
            },
            status_code=200,
        )


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=int(os.environ.get("PORT", 5000)))