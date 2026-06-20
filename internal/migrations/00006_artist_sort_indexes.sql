CREATE INDEX ON artists (updated_at DESC, id ASC);
CREATE INDEX ON artists (total_priority DESC, updated_at DESC, id ASC);

---- create above / drop below ----

DROP INDEX IF EXISTS artists_updated_at_id_idx;
DROP INDEX IF EXISTS artists_total_priority_updated_at_id_idx;
