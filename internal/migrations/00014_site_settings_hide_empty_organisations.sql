ALTER TABLE site_settings ADD COLUMN hide_empty_organisations BOOLEAN NOT NULL DEFAULT false;

---- create above / drop below ----

ALTER TABLE site_settings DROP COLUMN hide_empty_organisations;
