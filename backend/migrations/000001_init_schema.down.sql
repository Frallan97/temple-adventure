DROP TRIGGER IF EXISTS session_variables_updated_at ON session_variables;
DROP TRIGGER IF EXISTS game_sessions_updated_at ON game_sessions;
DROP FUNCTION IF EXISTS update_updated_at_column();
DROP TABLE IF EXISTS command_history;
DROP TABLE IF EXISTS session_variables;
DROP TABLE IF EXISTS session_inventory;
DROP TABLE IF EXISTS game_sessions;
