-- Composite index for type-specific queries ordered by name
CREATE INDEX ON organisations (type, name);

-- Allow multiple evidence sources per (artist, country) pair
CREATE TABLE artist_country_sources (
    artist_id INT  NOT NULL,
    code      TEXT NOT NULL,
    source_id INT  NOT NULL REFERENCES evidence_sources(id) ON DELETE CASCADE,
    PRIMARY KEY (artist_id, code, source_id),
    FOREIGN KEY (artist_id, code) REFERENCES artist_countries(artist_id, code) ON DELETE CASCADE
);

-- Migrate existing single source_id values
INSERT INTO artist_country_sources (artist_id, code, source_id)
SELECT artist_id, code, source_id
FROM artist_countries
WHERE source_id IS NOT NULL;

ALTER TABLE artist_countries DROP COLUMN source_id;

---- create above / drop below ----

ALTER TABLE artist_countries ADD COLUMN source_id INT REFERENCES evidence_sources(id) ON DELETE SET NULL;

UPDATE artist_countries ac
SET source_id = (
    SELECT source_id FROM artist_country_sources acs
    WHERE acs.artist_id = ac.artist_id AND acs.code = ac.code
    ORDER BY acs.source_id ASC
    LIMIT 1
);

DROP TABLE IF EXISTS artist_country_sources;

DROP INDEX IF EXISTS organisations_type_name_idx;
