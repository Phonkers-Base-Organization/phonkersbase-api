-- Unlabeled artists should default to -999 (matching the column default),
-- not 0 which is what COALESCE(SUM, 0) produces when no labels exist.
CREATE OR REPLACE FUNCTION calculate_total_priority()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE artists
    SET total_priority = (
        SELECT COALESCE(SUM(l.priority), -999)
        FROM artist_labels al
        JOIN labels l ON l.id = al.label_id
        WHERE al.artist_id = NEW.artist_id
    )
    WHERE id = NEW.artist_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Backfill artists that currently have total_priority=0 and no labels
UPDATE artists
SET total_priority = -999
WHERE total_priority = 0
  AND NOT EXISTS (SELECT 1 FROM artist_labels WHERE artist_id = artists.id);

---- create above / drop below ----

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
