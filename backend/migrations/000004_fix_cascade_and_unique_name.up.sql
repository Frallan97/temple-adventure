-- Fix: game_sessions.story_id should cascade on story delete
ALTER TABLE game_sessions DROP CONSTRAINT IF EXISTS game_sessions_story_id_fkey;
ALTER TABLE game_sessions ADD CONSTRAINT game_sessions_story_id_fkey
    FOREIGN KEY (story_id) REFERENCES stories(id) ON DELETE CASCADE;

-- Add unique constraint on story name
ALTER TABLE stories ADD CONSTRAINT stories_name_unique UNIQUE (name);
