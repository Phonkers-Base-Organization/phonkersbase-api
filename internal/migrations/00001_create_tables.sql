CREATE TABLE labels (
    id            SERIAL PRIMARY KEY,
    notion_id     TEXT        UNIQUE,
    name          TEXT        NOT NULL UNIQUE,                                                       
    original_name TEXT        NOT NULL,
    priority      INTEGER     NOT NULL DEFAULT 0,                                                    
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),     
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);                                                                                                   

CREATE TABLE artists (                                                                               
    id               SERIAL PRIMARY KEY,                  
    notion_id        TEXT        UNIQUE,
    name             TEXT        NOT NULL,
    link             TEXT,                    -- usually a Spotify artist URL
    spotify_id       TEXT,                                                                           
    avatar_url       TEXT,
    description_ua   TEXT,                    -- HTML, original language                             
    description_en   TEXT,                    -- HTML, English                                       
    total_priority   INTEGER     NOT NULL DEFAULT -999,  -- sum of related label priorities
    countries  TEXT[],                    -- list of country codes (ISO 3166-1 alpha-2) where the artist is popular or active               
    notion_created_at   TIMESTAMPTZ,             
    notion_edited_at    TIMESTAMPTZ,                                                        
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),                                             
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);                                                                                                   
                                                        
CREATE INDEX ON artists (name);                                                                      
CREATE INDEX ON artists (spotify_id);
CREATE INDEX ON artists (total_priority);                                                            
CREATE INDEX ON artists ((countries[1]));                

CREATE TABLE artist_labels (
    artist_id INTEGER NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
    label_id  INTEGER NOT NULL REFERENCES labels(id) ON DELETE CASCADE,                              
    PRIMARY KEY (artist_id, label_id)
);                                                                                                   
                                                        
CREATE INDEX ON artist_labels (label_id);      

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER artists_updated_at
    BEFORE UPDATE ON artists
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
  
CREATE TRIGGER labels_updated_at
    BEFORE UPDATE ON labels
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE OR REPLACE FUNCTION calculate_total_priority()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE artists
    SET total_priority = (
        SELECT COALESCE(SUM(l.priority), 0)
        FROM artist_labels al
        JOIN labels l ON l.id = al.label_id
        WHERE al.artist_id = NEW.artist_id
    )
    WHERE id = NEW.artist_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER total_priority_calculation
    AFTER INSERT OR UPDATE ON artist_labels
    FOR EACH ROW
    EXECUTE FUNCTION calculate_total_priority();

---- create above / drop below ----

DROP TABLE IF EXISTS artist_labels;
DROP TABLE IF EXISTS artists;
DROP TABLE IF EXISTS labels;

DROP FUNCTION IF EXISTS set_updated_at;
DROP FUNCTION IF EXISTS calculate_total_priority;
DROP TRIGGER IF EXISTS artists_updated_at ON artists;
DROP TRIGGER IF EXISTS labels_updated_at ON labels;
DROP TRIGGER IF EXISTS total_priority_calculation ON artist_labels;