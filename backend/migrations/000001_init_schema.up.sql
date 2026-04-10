CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS game_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    current_room_id VARCHAR(64) NOT NULL DEFAULT 'entrance',
    turn_number INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS session_inventory (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id UUID NOT NULL REFERENCES game_sessions(id) ON DELETE CASCADE,
    item_id VARCHAR(64) NOT NULL,
    acquired_at_turn INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(session_id, item_id)
);

CREATE TABLE IF NOT EXISTS session_variables (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id UUID NOT NULL REFERENCES game_sessions(id) ON DELETE CASCADE,
    var_key VARCHAR(128) NOT NULL,
    var_type VARCHAR(16) NOT NULL,
    val_bool BOOLEAN,
    val_int INTEGER,
    val_string TEXT,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(session_id, var_key)
);

CREATE TABLE IF NOT EXISTS command_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id UUID NOT NULL REFERENCES game_sessions(id) ON DELETE CASCADE,
    turn_number INTEGER NOT NULL,
    raw_input TEXT NOT NULL,
    parsed_verb VARCHAR(64),
    parsed_target VARCHAR(128),
    room_id VARCHAR(64) NOT NULL,
    response_text TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_session_inventory_session ON session_inventory(session_id);
CREATE INDEX idx_session_variables_session ON session_variables(session_id);
CREATE INDEX idx_session_variables_lookup ON session_variables(session_id, var_key);
CREATE INDEX idx_command_history_session ON command_history(session_id);
CREATE INDEX idx_command_history_turn ON command_history(session_id, turn_number);
CREATE INDEX idx_game_sessions_status ON game_sessions(status);

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER game_sessions_updated_at BEFORE UPDATE ON game_sessions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER session_variables_updated_at BEFORE UPDATE ON session_variables
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
