import os
import sqlite3
from fastapi import FastAPI
from fastapi.responses import JSONResponse, Response

app = FastAPI()
DATABASE = os.environ.get("DB_PATH", "app.db")


def get_db():
    db = sqlite3.connect(DATABASE)
    db.row_factory = sqlite3.Row
    db.execute("PRAGMA foreign_keys = ON;")
    return db


@app.delete("/api/customers/{c_custkey}")
def delete_customer(c_custkey: int):
    db = get_db()
    try:
        existing = db.execute(
            "SELECT c_custkey FROM customer WHERE c_custkey = ?", (c_custkey,)
        ).fetchone()
        if existing is None:
            return JSONResponse({"error": "Customer not found"}, status_code=404)
        try:
            db.execute("DELETE FROM customer WHERE c_custkey = ?", (c_custkey,))
            db.commit()
        except sqlite3.IntegrityError:
            return JSONResponse(
                {"error": "Cannot delete customer with existing orders"},
                status_code=409,
            )
        return Response(status_code=204)
    finally:
        db.close()


@app.get("/api/dashboard")
def dashboard():
    """
    Return dashboard based on all existed orders
    """
    db = get_db()
    try:
        row = db.execute(
            "SELECT COUNT(*) AS total_orders, "
            "       COALESCE(SUM(o_totalprice), 0) AS total_revenue "
            "FROM orders"
        ).fetchone()
        return JSONResponse(dict(row), status_code=200)
    finally:
        db.close()


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=int(os.environ.get("PORT", 5000)))