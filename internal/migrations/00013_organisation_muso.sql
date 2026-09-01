CREATE TABLE organisation_muso (
    id                  BIGSERIAL PRIMARY KEY,
    organisation_id     INT NOT NULL UNIQUE REFERENCES organisations(id) ON DELETE CASCADE,
    muso_profile_id     TEXT NOT NULL,
    -- Reported by muso search and used as the roster change-detection signal.
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

CREATE TABLE organisation_artists (
    organisation_id INT NOT NULL REFERENCES organisations(id) ON DELETE CASCADE,
    artist_id       INT NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
    muso_profile_id TEXT NOT NULL,
    PRIMARY KEY (organisation_id, artist_id)
);

CREATE TRIGGER organisation_muso_updated_at
    BEFORE UPDATE ON organisation_muso
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- The organisation_artists index serves the distribution join; the
-- organisation_muso index serves oldest-first nightly sweep ordering.
CREATE INDEX idx_organisation_artists_organisation_id ON organisation_artists (organisation_id);
CREATE INDEX idx_organisation_muso_last_checked_at ON organisation_muso (last_checked_at);

---- create above / drop below ----

DROP TABLE IF EXISTS organisation_artists;
DROP TABLE IF EXISTS organisation_muso;
