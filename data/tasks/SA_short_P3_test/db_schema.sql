CREATE TABLE orders (
    o_orderkey       INTEGER PRIMARY KEY,
    o_totalprice     REAL,
    o_orderdate      TEXT,
    o_latest_shipdate TEXT,
    o_comment        TEXT
);

CREATE TABLE lineitem (
    l_orderkey      INTEGER,
    l_linenumber    INTEGER,
    l_extendedprice REAL,
    l_discount      REAL,
    l_tax           REAL,
    l_shipdate      TEXT,
    l_comment       TEXT,
    PRIMARY KEY (l_orderkey, l_linenumber),
    FOREIGN KEY (l_orderkey) REFERENCES orders(o_orderkey)
);
