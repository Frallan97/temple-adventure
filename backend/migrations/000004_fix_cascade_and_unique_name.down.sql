ALTER TABLE stories DROP CONSTRAINT IF EXISTS stories_name_unique;

ALTER TABLE game_sessions DROP CONSTRAINT IF EXISTS game_sessions_story_id_fkey;
ALTER TABLE game_sessions ADD CONSTRAINT game_sessions_story_id_fkey
    FOREIGN KEY (story_id) REFERENCES stories(id);
