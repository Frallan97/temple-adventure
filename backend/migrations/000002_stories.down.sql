ALTER TABLE game_sessions DROP COLUMN IF EXISTS story_id;

DROP TABLE IF EXISTS story_puzzles;
DROP TABLE IF EXISTS story_items;
DROP TABLE IF EXISTS story_rooms;
DROP TABLE IF EXISTS stories;
