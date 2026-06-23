ALTER TABLE suggestions DROP COLUMN status;

---- create above / drop below ----

ALTER TABLE suggestions ADD COLUMN status TEXT NOT NULL DEFAULT 'pending';
