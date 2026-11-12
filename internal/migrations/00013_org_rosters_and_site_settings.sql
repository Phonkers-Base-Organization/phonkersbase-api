ALTER TABLE organisations ADD COLUMN roster_size INT NOT NULL DEFAULT 0;

CREATE TABLE organisation_artists (
    organisation_id INT NOT NULL REFERENCES organisations(id) ON DELETE CASCADE,
    artist_id       INT NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
    PRIMARY KEY (organisation_id, artist_id)
);

CREATE INDEX idx_organisation_artists_organisation_id ON organisation_artists (organisation_id);

CREATE TABLE site_settings (
    id                                BOOLEAN PRIMARY KEY DEFAULT true CHECK (id),
    show_organisation_recommendations BOOLEAN NOT NULL DEFAULT false,
    updated_at                        TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO site_settings DEFAULT VALUES;

CREATE TRIGGER site_settings_updated_at
    BEFORE UPDATE ON site_settings
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

---- create above / drop below ----

DROP TABLE IF EXISTS site_settings;

DROP TABLE IF EXISTS organisation_artists;

ALTER TABLE organisations DROP COLUMN roster_size;
