CREATE TABLE change_history (
    id              BIGSERIAL PRIMARY KEY,
    entity_type     TEXT NOT NULL,
    entity_id       TEXT NOT NULL,
    entity_name     TEXT NOT NULL,
    action          TEXT NOT NULL,
    editor_id       TEXT NOT NULL DEFAULT '',
    editor_username TEXT NOT NULL,
    old_data        JSONB,
    new_data        JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_change_history_created_at ON change_history (created_at DESC);
CREATE INDEX idx_change_history_entity_type_created_at ON change_history (entity_type, created_at DESC);
CREATE INDEX idx_change_history_entity_name_trgm ON change_history USING gin (entity_name gin_trgm_ops);

---- create above / drop below ----

DROP TABLE IF EXISTS change_history;
