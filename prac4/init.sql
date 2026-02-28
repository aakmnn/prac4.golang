CREATE TABLE IF NOT EXISTS movies (
  id SERIAL PRIMARY KEY,
  title TEXT NOT NULL
);

-- optional seed data (can remove if you want empty DB)
INSERT INTO movies (title)
SELECT 'Sample Movie'
WHERE NOT EXISTS (SELECT 1 FROM movies);
