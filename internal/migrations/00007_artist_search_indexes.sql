CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX idx_artists_name_trgm ON artists USING gin (name gin_trgm_ops);
CREATE INDEX idx_artists_description_ua_trgm ON artists USING gin (description_ua gin_trgm_ops);
CREATE INDEX idx_artists_description_en_trgm ON artists USING gin (description_en gin_trgm_ops);
CREATE INDEX idx_artists_link_trgm ON artists USING gin (split_part(link, '?', 1) gin_trgm_ops);

DROP INDEX IF EXISTS artists_spotify_id_idx;
CREATE INDEX idx_artists_spotify_id_hash ON artists USING hash (spotify_id);

---- create above / drop below ----

DROP INDEX IF EXISTS idx_artists_spotify_id_hash;
CREATE INDEX ON artists (spotify_id);

DROP INDEX IF EXISTS idx_artists_link_trgm;
DROP INDEX IF EXISTS idx_artists_description_en_trgm;
DROP INDEX IF EXISTS idx_artists_description_ua_trgm;
DROP INDEX IF EXISTS idx_artists_name_trgm;

DROP EXTENSION IF EXISTS pg_trgm;
