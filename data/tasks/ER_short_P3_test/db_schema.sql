CREATE TABLE supplier (
    s_suppkey INTEGER PRIMARY KEY,
    s_name    TEXT,
    s_status  TEXT
);

CREATE TABLE product (
    p_productkey  INTEGER PRIMARY KEY,
    p_name        TEXT,
    p_suppkey     INTEGER,
    p_retailprice REAL,
    FOREIGN KEY (p_suppkey) REFERENCES supplier(s_suppkey)
);
