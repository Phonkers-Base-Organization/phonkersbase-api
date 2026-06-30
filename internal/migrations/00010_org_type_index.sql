-- Composite index for type-specific queries ordered by name
CREATE INDEX ON organisations (type, name);

---- create above / drop below ----

DROP INDEX IF EXISTS organisations_type_name_idx;
