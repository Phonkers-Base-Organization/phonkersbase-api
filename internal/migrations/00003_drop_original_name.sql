ALTER TABLE labels DROP COLUMN original_name;

---- create above / drop below ----

ALTER TABLE labels ADD COLUMN original_name TEXT NOT NULL DEFAULT '';
