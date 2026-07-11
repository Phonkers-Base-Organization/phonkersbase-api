-- The /history endpoint filters by entity_id, editor_username, and action,
-- always ordered by (created_at DESC, id DESC) with LIMIT/OFFSET. Each index
-- embeds that ordering so filtered pages are served as top-N index scans
-- instead of full scans + sorts as the table grows.
CREATE INDEX idx_change_history_entity_id ON change_history (entity_id, created_at DESC, id DESC);
CREATE INDEX idx_change_history_editor_username ON change_history (editor_username, created_at DESC, id DESC);
CREATE INDEX idx_change_history_action ON change_history (action, created_at DESC, id DESC);

---- create above / drop below ----

DROP INDEX IF EXISTS idx_change_history_entity_id;
DROP INDEX IF EXISTS idx_change_history_editor_username;
DROP INDEX IF EXISTS idx_change_history_action;
