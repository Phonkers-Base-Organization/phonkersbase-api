-- Serves the exact, case-insensitive name lookup used by multi-term artist search
-- (`lower(name) = ANY(...)`). Multi-term search previously matched every term as an ILIKE
-- substring across name/description/link, which the planner resolved as a sequential scan
-- costing ~10ms per term; this index makes it one bitmap index scan regardless of term count.
CREATE INDEX idx_artists_lower_name ON artists (lower(name));

---- create above / drop below ----

DROP INDEX IF EXISTS idx_artists_lower_name;
