ALTER TABLE organisation_artists ALTER COLUMN muso_profile_id DROP NOT NULL;

---- create above / drop below ----

DELETE FROM organisation_artists WHERE muso_profile_id IS NULL;
ALTER TABLE organisation_artists ALTER COLUMN muso_profile_id SET NOT NULL;
