CREATE TABLE organisations (
    id             SERIAL PRIMARY KEY,
    name           TEXT        NOT NULL,
    link           TEXT,
    origin         TEXT        NOT NULL,
    description_uk TEXT,
    description_en TEXT,
    notes          TEXT,
    type           TEXT        NOT NULL,
    recommendation TEXT        NOT NULL
        CONSTRAINT organisations_recommendation_check CHECK (recommendation IN (
            'Не використовуй',
            'Не слухай це',
            'Будь обережний',
            'Можеш використовувати',
            'Можеш слухати'
        )),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ON organisations (type);
CREATE INDEX ON organisations (name);

CREATE TRIGGER organisations_updated_at
    BEFORE UPDATE ON organisations
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

ALTER TABLE evidence_sources DROP COLUMN name_uk;

---- create above / drop below ----

ALTER TABLE evidence_sources ADD COLUMN name_uk TEXT NOT NULL DEFAULT '';
DROP TABLE IF EXISTS organisations;
