-- Fix incorrect country codes produced by the UPPER(SUBSTRING(name, 1, 2)) fallback
-- in migration 00002. Any country name not in the explicit CASE list got its first
-- two letters uppercased, producing invalid ISO codes:
--
--   el_salvador → 'EL'  (correct ISO-3166-1: SV)
--   puerto_rico → 'PU'  (correct ISO-3166-1: PR)

UPDATE artist_countries SET code = 'SV' WHERE code = 'EL';
UPDATE artist_countries SET code = 'PR' WHERE code = 'PU';

---- create above / drop below ----

UPDATE artist_countries SET code = 'EL' WHERE code = 'SV';
UPDATE artist_countries SET code = 'PU' WHERE code = 'PR';
