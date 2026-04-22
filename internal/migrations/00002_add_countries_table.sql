CREATE TABLE countries (
    id            SERIAL PRIMARY KEY,
    name          TEXT        NOT NULL UNIQUE,
    original_name TEXT        NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER countries_updated_at
    BEFORE UPDATE ON countries
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

---- create above / drop below ----

DROP TRIGGER IF EXISTS countries_updated_at ON countries;
DROP TABLE IF EXISTS countries;
