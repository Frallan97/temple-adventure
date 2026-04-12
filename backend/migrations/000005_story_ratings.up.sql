CREATE TABLE IF NOT EXISTS story_ratings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    story_id UUID NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
    session_id UUID NOT NULL REFERENCES game_sessions(id) ON DELETE CASCADE,
    rating SMALLINT NOT NULL CHECK (rating >= 1 AND rating <= 5),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (session_id)
);

CREATE INDEX idx_story_ratings_story_id ON story_ratings(story_id);
