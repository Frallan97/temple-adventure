CREATE TABLE story_npcs (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    story_id    UUID NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
    npc_id      VARCHAR(64) NOT NULL,
    name        VARCHAR(128) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    aliases     JSONB NOT NULL DEFAULT '[]',
    room        VARCHAR(64) NOT NULL,
    dialogue    JSONB NOT NULL DEFAULT '[]',
    movement    JSONB NOT NULL DEFAULT '[]',
    conditional_descriptions JSONB NOT NULL DEFAULT '[]',
    UNIQUE(story_id, npc_id)
);
CREATE INDEX idx_story_npcs_story_id ON story_npcs(story_id);
