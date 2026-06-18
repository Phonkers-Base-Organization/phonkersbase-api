ALTER TABLE evidence_sources ADD COLUMN name_uk TEXT NOT NULL DEFAULT '';
ALTER TABLE evidence_sources ADD COLUMN name_en TEXT NOT NULL DEFAULT '';

UPDATE evidence_sources SET name_uk = name;

UPDATE evidence_sources SET name_en = CASE id
  WHEN 1  THEN 'Needs clarification'
  WHEN 2  THEN 'Open info from the internet'
  WHEN 3  THEN 'Spotify description'
  WHEN 4  THEN 'Track covers/titles'
  WHEN 5  THEN 'Spotify full name'
  WHEN 6  THEN 'SoundCloud description/country'
  WHEN 7  THEN 'SoundCloud comments'
  WHEN 8  THEN 'Bandcamp country'
  WHEN 9  THEN 'Language in social media'
  WHEN 10 THEN 'Instagram description'
  WHEN 11 THEN 'Instagram geo'
  WHEN 12 THEN 'Instagram posts'
  WHEN 13 THEN 'YouTube description/country'
  WHEN 14 THEN 'TikTok description'
  WHEN 15 THEN 'TikTok geo'
  WHEN 16 THEN 'TikTok posts'
  WHEN 17 THEN 'Genius description'
  WHEN 18 THEN 'VK group description'
  WHEN 19 THEN 'VK account description/country'
  WHEN 20 THEN 'VK posts'
  WHEN 21 THEN 'Telegram description'
  WHEN 22 THEN 'Telegram posts'
  WHEN 23 THEN 'Twitter description/country'
  WHEN 24 THEN 'Twitter geo'
  WHEN 25 THEN 'Twitter posts'
  WHEN 26 THEN 'Facebook description/country'
  WHEN 27 THEN 'Facebook geo'
  WHEN 28 THEN 'Facebook posts'
  WHEN 29 THEN 'Discord description'
  WHEN 30 THEN 'Discord messages'
  WHEN 31 THEN 'By activity'
  WHEN 32 THEN 'Artist''s second account'
  WHEN 33 THEN 'NCS fandom'
  WHEN 34 THEN 'BeatStars profile'
  WHEN 35 THEN 'Beatport description'
  WHEN 36 THEN 'Info from the label'
  WHEN 37 THEN 'Linktree link country'
  WHEN 38 THEN 'LinkedIn country'
  WHEN 39 THEN 'Viberate.com country'
  WHEN 40 THEN 'Info from artist/acquaintances'
  WHEN 41 THEN 'Monstercat wiki'
  WHEN 42 THEN 'Reddit comments'
  WHEN 43 THEN 'Wikitubia description'
  WHEN 44 THEN 'Other sources'
  WHEN 45 THEN 'Guns.lol country'
  WHEN 46 THEN 'Threads posts/comments'
  ELSE name
END;

---- create above / drop below ----

ALTER TABLE evidence_sources DROP COLUMN IF EXISTS name_uk;
ALTER TABLE evidence_sources DROP COLUMN IF EXISTS name_en;
