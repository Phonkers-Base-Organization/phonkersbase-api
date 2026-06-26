UPDATE organisations SET recommendation = 'Не використовуй'      WHERE recommendation ILIKE 'Не використовуй%';
UPDATE organisations SET recommendation = 'Не слухай це'         WHERE recommendation ILIKE 'Не слухай це%';
UPDATE organisations SET recommendation = 'Будь обережний'       WHERE recommendation ILIKE 'Будь обережний%';
UPDATE organisations SET recommendation = 'Можеш використовувати' WHERE recommendation ILIKE 'Можеш використовувати%';
UPDATE organisations SET recommendation = 'Можеш слухати'        WHERE recommendation ILIKE 'Можеш слухати%';

ALTER TABLE organisations
    ADD CONSTRAINT organisations_recommendation_check
    CHECK (recommendation IN (
        'Не використовуй',
        'Не слухай це',
        'Будь обережний',
        'Можеш використовувати',
        'Можеш слухати'
    ));

---- create above / drop below ----

ALTER TABLE organisations DROP CONSTRAINT IF EXISTS organisations_recommendation_check;
