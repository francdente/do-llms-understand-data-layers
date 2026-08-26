CREATE TABLE customer (
    c_custkey INTEGER PRIMARY KEY,
    c_name    TEXT,
    c_email   TEXT
);

CREATE TABLE orders (
    o_orderkey    INTEGER PRIMARY KEY,
    o_custkey     INTEGER,
    o_totalprice  REAL,
    o_orderdate   TEXT,
    o_comment     TEXT,
    FOREIGN KEY (o_custkey) REFERENCES customer(c_custkey)
);
