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
