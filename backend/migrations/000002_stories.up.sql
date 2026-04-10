-- Story metadata
CREATE TABLE stories (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        VARCHAR(128) NOT NULL,
    slug        VARCHAR(128) NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    author      VARCHAR(128) NOT NULL DEFAULT 'Anonymous',
    start_room  VARCHAR(64) NOT NULL DEFAULT 'entrance',
    is_published BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER update_stories_updated_at
    BEFORE UPDATE ON stories
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Rooms belonging to a story
CREATE TABLE story_rooms (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    story_id    UUID NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
    room_id     VARCHAR(64) NOT NULL,
    name        VARCHAR(128) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    connections JSONB NOT NULL DEFAULT '{}',
    items       JSONB NOT NULL DEFAULT '[]',
    puzzles     JSONB NOT NULL DEFAULT '[]',
    conditional_descriptions JSONB NOT NULL DEFAULT '[]',
    hints       JSONB NOT NULL DEFAULT '[]',
    UNIQUE(story_id, room_id)
);

CREATE INDEX idx_story_rooms_story_id ON story_rooms(story_id);

-- Items belonging to a story
CREATE TABLE story_items (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    story_id    UUID NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
    item_id     VARCHAR(64) NOT NULL,
    name        VARCHAR(128) NOT NULL,
    aliases     JSONB NOT NULL DEFAULT '[]',
    description TEXT NOT NULL DEFAULT '',
    portable    BOOLEAN NOT NULL DEFAULT false,
    interactions JSONB NOT NULL DEFAULT '[]',
    conditional_descriptions JSONB NOT NULL DEFAULT '[]',
    UNIQUE(story_id, item_id)
);

CREATE INDEX idx_story_items_story_id ON story_items(story_id);

-- Puzzles belonging to a story
CREATE TABLE story_puzzles (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    story_id    UUID NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
    puzzle_id   VARCHAR(64) NOT NULL,
    name        VARCHAR(128) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    steps       JSONB NOT NULL DEFAULT '[]',
    timed_window JSONB,
    failure_effects JSONB NOT NULL DEFAULT '[]',
    failure_text TEXT NOT NULL DEFAULT '',
    completion_text TEXT NOT NULL DEFAULT '',
    UNIQUE(story_id, puzzle_id)
);

CREATE INDEX idx_story_puzzles_story_id ON story_puzzles(story_id);

-- Link game sessions to a story
ALTER TABLE game_sessions ADD COLUMN story_id UUID REFERENCES stories(id);
CREATE INDEX idx_game_sessions_story_id ON game_sessions(story_id);
