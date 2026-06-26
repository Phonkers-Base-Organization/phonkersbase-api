CREATE TABLE organisations (
    id             SERIAL PRIMARY KEY,
    name           TEXT        NOT NULL,
    link           TEXT,
    origin         TEXT        NOT NULL,
    info           TEXT,
    type           TEXT        NOT NULL,
    recommendation TEXT        NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ON organisations (type);
CREATE INDEX ON organisations (name);

CREATE TRIGGER organisations_updated_at
    BEFORE UPDATE ON organisations
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

---- create above / drop below ----

DROP TABLE IF EXISTS organisations;
