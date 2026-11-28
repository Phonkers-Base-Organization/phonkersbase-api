CREATE TABLE organisation_countries (
    organisation_id INT NOT NULL REFERENCES organisations(id) ON DELETE CASCADE,
    code            TEXT NOT NULL,
    position        SMALLINT NOT NULL DEFAULT 0,
    PRIMARY KEY (organisation_id, code)
);

CREATE TABLE organisation_country_sources (
    organisation_id INT NOT NULL,
    code            TEXT NOT NULL,
    source_id       INT NOT NULL REFERENCES evidence_sources(id) ON DELETE CASCADE,
    PRIMARY KEY (organisation_id, code, source_id),
    FOREIGN KEY (organisation_id, code) REFERENCES organisation_countries(organisation_id, code) ON DELETE CASCADE
);

INSERT INTO organisation_countries (organisation_id, code, position)
SELECT id, origin, 0 FROM organisations;

ALTER TABLE organisations DROP COLUMN origin;

ALTER TABLE site_settings ADD COLUMN hide_distributors_and_cults BOOLEAN NOT NULL DEFAULT true;

---- create above / drop below ----

ALTER TABLE site_settings DROP COLUMN hide_distributors_and_cults;

ALTER TABLE organisations ADD COLUMN origin TEXT;

UPDATE organisations o SET origin = (
    SELECT oc.code FROM organisation_countries oc
    WHERE oc.organisation_id = o.id AND oc.position = 0
);

UPDATE organisations SET origin = '' WHERE origin IS NULL;

ALTER TABLE organisations ALTER COLUMN origin SET NOT NULL;

DROP TABLE organisation_country_sources;
DROP TABLE organisation_countries;
