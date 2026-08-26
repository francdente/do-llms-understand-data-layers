CREATE TABLE product (
    p_productkey  INTEGER PRIMARY KEY,
    p_name        TEXT,
    p_status      TEXT,
    p_retailprice REAL
);

CREATE TABLE inventory (
    i_productkey INTEGER,
    i_warehouse  TEXT,
    i_quantity   INTEGER,
    PRIMARY KEY (i_productkey, i_warehouse),
    FOREIGN KEY (i_productkey) REFERENCES product(p_productkey)
);
