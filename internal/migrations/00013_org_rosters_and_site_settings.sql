CREATE TABLE organisation_muso (
    id                  BIGSERIAL PRIMARY KEY,
    organisation_id     INT NOT NULL UNIQUE REFERENCES organisations(id) ON DELETE CASCADE,
    muso_profile_id     TEXT NOT NULL,
    collaborators_count INT NOT NULL DEFAULT 0,
    credit_count        INT NOT NULL DEFAULT 0,
    roster_size         INT NOT NULL DEFAULT 0,
    last_checked_at     TIMESTAMPTZ,
    last_synced_at      TIMESTAMPTZ,
    sync_status         TEXT NOT NULL DEFAULT 'pending'
        CONSTRAINT organisation_muso_sync_status_check CHECK (sync_status IN (
            'pending',
            'ok',
            'error',
            'not_found'
        )),
    sync_error          TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- muso_profile_id is NULL on artists an admin added by hand.
CREATE TABLE organisation_artists (
    organisation_id INT NOT NULL REFERENCES organisations(id) ON DELETE CASCADE,
    artist_id       INT NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
    muso_profile_id TEXT,
    PRIMARY KEY (organisation_id, artist_id)
);

CREATE TRIGGER organisation_muso_updated_at
    BEFORE UPDATE ON organisation_muso
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

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
DROP TABLE IF EXISTS organisation_muso;
