DROP TABLE IF EXISTS countries;

UPDATE artists SET countries = ARRAY(
  SELECT CASE c
    WHEN 'ukraine' THEN 'UA'
    WHEN 'russian_federation' THEN 'RU'
    WHEN 'ruzzia' THEN 'RU'
    WHEN 'belarus' THEN 'BY'
    WHEN 'germany' THEN 'DE'
    WHEN 'poland' THEN 'PL'
    WHEN 'usa' THEN 'US'
    WHEN 'united_states_of_america' THEN 'US'
    WHEN 'united_kingdom' THEN 'GB'
    WHEN 'france' THEN 'FR'
    WHEN 'italy' THEN 'IT'
    WHEN 'spain' THEN 'ES'
    WHEN 'czechia' THEN 'CZ'
    WHEN 'sweden' THEN 'SE'
    WHEN 'norway' THEN 'NO'
    WHEN 'finland' THEN 'FI'
    WHEN 'denmark' THEN 'DK'
    WHEN 'netherlands' THEN 'NL'
    WHEN 'hungary' THEN 'HU'
    WHEN 'romania' THEN 'RO'
    WHEN 'bulgaria' THEN 'BG'
    WHEN 'serbia' THEN 'RS'
    WHEN 'croatia' THEN 'HR'
    WHEN 'moldova' THEN 'MD'
    WHEN 'georgia' THEN 'GE'
    WHEN 'armenia' THEN 'AM'
    WHEN 'azerbaijan' THEN 'AZ'
    WHEN 'kazakhstan' THEN 'KZ'
    WHEN 'uzbekistan' THEN 'UZ'
    WHEN 'latvia' THEN 'LV'
    WHEN 'lithuania' THEN 'LT'
    WHEN 'estonia' THEN 'EE'
    ELSE UPPER(SUBSTRING(c, 1, 2))
  END
  FROM unnest(countries) AS c
) WHERE countries IS NOT NULL AND array_length(countries, 1) > 0;

ALTER TABLE artists ADD COLUMN IF NOT EXISTS primary_country TEXT;
ALTER TABLE artists ADD COLUMN IF NOT EXISTS evidence_url TEXT;
ALTER TABLE artists ADD COLUMN IF NOT EXISTS notes TEXT;
ALTER TABLE artists ADD COLUMN IF NOT EXISTS sources JSONB NOT NULL DEFAULT '[]';

CREATE TABLE users (
    id            SERIAL PRIMARY KEY,
    username      TEXT        NOT NULL UNIQUE,
    password_hash TEXT        NOT NULL,
    role          TEXT        NOT NULL DEFAULT 'EDITOR',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE suggestions (
    id            SERIAL PRIMARY KEY,
    name          TEXT        NOT NULL,
    link          TEXT,
    countries     TEXT[],
    listen_labels TEXT[],
    evidence      TEXT,
    description   TEXT,
    status        TEXT        NOT NULL DEFAULT 'pending',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE feedbacks (
    id         SERIAL PRIMARY KEY,
    type       TEXT        NOT NULL,
    text       TEXT        NOT NULL,
    email      TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER suggestions_updated_at
    BEFORE UPDATE ON suggestions
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

---- create above / drop below ----

DROP TRIGGER IF EXISTS users_updated_at ON users;
DROP TRIGGER IF EXISTS suggestions_updated_at ON suggestions;

DROP TABLE IF EXISTS feedbacks;
DROP TABLE IF EXISTS suggestions;
DROP TABLE IF EXISTS users;

ALTER TABLE artists DROP COLUMN IF EXISTS sources;
ALTER TABLE artists DROP COLUMN IF EXISTS notes;
ALTER TABLE artists DROP COLUMN IF EXISTS evidence_url;
ALTER TABLE artists DROP COLUMN IF EXISTS primary_country;

CREATE TABLE countries (
    id            SERIAL PRIMARY KEY,
    name          TEXT        NOT NULL UNIQUE,
    original_name TEXT        NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER countries_updated_at
    BEFORE UPDATE ON countries
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
