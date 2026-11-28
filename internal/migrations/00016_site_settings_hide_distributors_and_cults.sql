ALTER TABLE site_settings ADD COLUMN hide_distributors_and_cults BOOLEAN NOT NULL DEFAULT true;

---- create above / drop below ----

ALTER TABLE site_settings DROP COLUMN hide_distributors_and_cults;
