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

-- The /history endpoint also filters by entity_id, editor_username, and action,
-- always ordered by (created_at DESC, id DESC) with LIMIT/OFFSET. Each index
-- embeds that ordering so filtered pages are served as top-N index scans
-- instead of full scans + sorts as the table grows.
CREATE INDEX idx_change_history_entity_id ON change_history (entity_id, created_at DESC, id DESC);
CREATE INDEX idx_change_history_editor_username ON change_history (editor_username, created_at DESC, id DESC);
CREATE INDEX idx_change_history_action ON change_history (action, created_at DESC, id DESC);

---- create above / drop below ----

DROP TABLE IF EXISTS change_history;
