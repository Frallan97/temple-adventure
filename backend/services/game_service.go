package services

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"temple-adventure/engine"
	"temple-adventure/models"

	"github.com/google/uuid"
)

type GameService struct {
	db         *sql.DB
	storyRepo  *StoryRepository
	cache      *engine.EngineCache
	// fallbackEngine is used for sessions without a story_id (legacy)
	fallbackEngine *engine.Engine
}

func NewGameService(db *sql.DB, fallbackEngine *engine.Engine) *GameService {
	return &GameService{
		db:             db,
		storyRepo:      NewStoryRepository(db),
		cache:          engine.NewEngineCache(),
		fallbackEngine: fallbackEngine,
	}
}

func (s *GameService) Cache() *engine.EngineCache {
	return s.cache
}

// getEngine returns the engine for a story, loading and caching as needed.
func (s *GameService) getEngine(ctx context.Context, storyID uuid.UUID) (*engine.Engine, string, error) {
	if eng, ok := s.cache.Get(storyID); ok {
		story, err := s.storyRepo.GetByID(ctx, storyID)
		if err != nil {
			return nil, "", err
		}
		return eng, story.StartRoom, nil
	}

	story, err := s.storyRepo.GetByID(ctx, storyID)
	if err != nil {
		return nil, "", fmt.Errorf("loading story: %w", err)
	}

	world, err := s.storyRepo.LoadWorldDefinition(ctx, storyID)
	if err != nil {
		return nil, "", fmt.Errorf("loading world definition: %w", err)
	}

	eng := engine.NewEngineFromWorld(world)
	s.cache.Set(storyID, eng)
	return eng, story.StartRoom, nil
}

func (s *GameService) CreateGame(ctx context.Context, storyID uuid.UUID) (*models.CreateGameResponse, error) {
	eng, startRoom, err := s.getEngine(ctx, storyID)
	if err != nil {
		return nil, fmt.Errorf("getting engine for story %s: %w", storyID, err)
	}

	state := eng.World.NewWorldState("", startRoom)

	var session models.GameSession
	err = s.db.QueryRowContext(ctx,
		`INSERT INTO game_sessions (current_room_id, turn_number, status, story_id) VALUES ($1, $2, $3, $4)
		 RETURNING id, current_room_id, turn_number, status, created_at, updated_at`,
		state.CurrentRoom, state.TurnNumber, state.Status, storyID,
	).Scan(&session.ID, &session.CurrentRoomID, &session.TurnNumber, &session.Status, &session.CreatedAt, &session.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating game session: %w", err)
	}

	state.SessionID = session.ID.String()
	roomDesc := eng.GetRoomDescription(state)
	roomName := eng.World.Rooms[state.CurrentRoom].Name

	return &models.CreateGameResponse{
		ID:          session.ID,
		StoryID:     storyID,
		RoomName:    roomName,
		Description: roomDesc,
		TurnNumber:  session.TurnNumber,
		Inventory:   []models.ItemInfoResponse{},
	}, nil
}

func (s *GameService) ProcessCommand(ctx context.Context, sessionID uuid.UUID, input string) (*models.CommandResponse, error) {
	state, eng, err := s.loadWorldState(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("[session:%s] loading state: %w", sessionID, err)
	}

	result := eng.ProcessCommand(state, input)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("[session:%s] beginning transaction: %w", sessionID, err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx,
		`UPDATE game_sessions SET current_room_id = $1, turn_number = $2, status = $3 WHERE id = $4`,
		state.CurrentRoom, state.TurnNumber, state.Status, sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("[session:%s] updating session: %w", sessionID, err)
	}

	if err := s.syncInventory(ctx, tx, sessionID, state); err != nil {
		return nil, fmt.Errorf("[session:%s] syncing inventory: %w", sessionID, err)
	}

	if err := s.syncVariables(ctx, tx, sessionID, state); err != nil {
		return nil, fmt.Errorf("[session:%s] syncing variables: %w", sessionID, err)
	}

	cmd := eng.ParseCommand(input)
	_, err = tx.ExecContext(ctx,
		`INSERT INTO command_history (session_id, turn_number, raw_input, parsed_verb, parsed_target, room_id, response_text)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		sessionID, state.TurnNumber, input, cmd.Verb, cmd.Target, state.CurrentRoom, result.Text,
	)
	if err != nil {
		return nil, fmt.Errorf("[session:%s] saving command history: %w", sessionID, err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("[session:%s] committing transaction: %w", sessionID, err)
	}

	roomName := eng.World.Rooms[state.CurrentRoom].Name
	inventory := eng.GetInventory(state)

	var choices []models.ChoiceResponse
	for _, c := range result.Choices {
		choices = append(choices, models.ChoiceResponse{Index: c.Index, Text: c.Text})
	}

	return &models.CommandResponse{
		Text:        result.Text,
		RoomName:    roomName,
		RoomChanged: result.RoomChanged,
		TurnNumber:  result.TurnNumber,
		GameOver:    result.GameOver,
		GameStatus:  result.GameStatus,
		Inventory:   toItemInfoResponses(inventory),
		Choices:     choices,
		EndingID:    result.EndingID,
		EndingTitle: result.EndingTitle,
	}, nil
}

func (s *GameService) GetGameState(ctx context.Context, sessionID uuid.UUID) (*models.GameStateResponse, error) {
	state, eng, err := s.loadWorldState(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	roomName := eng.World.Rooms[state.CurrentRoom].Name
	roomDesc := eng.GetRoomDescription(state)
	inventory := eng.GetInventory(state)

	return &models.GameStateResponse{
		ID:          sessionID,
		RoomName:    roomName,
		Description: roomDesc,
		TurnNumber:  state.TurnNumber,
		Status:      state.Status,
		Inventory:   toItemInfoResponses(inventory),
	}, nil
}

func (s *GameService) GetHistory(ctx context.Context, sessionID uuid.UUID, limit, offset int) (*models.HistoryResponse, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM game_sessions WHERE id = $1)`, sessionID).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("checking session: %w", err)
	}
	if !exists {
		return nil, models.NewNotFoundError("game session")
	}

	var total int
	err = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM command_history WHERE session_id = $1`, sessionID).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("counting history: %w", err)
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, session_id, turn_number, raw_input, parsed_verb, parsed_target, room_id, response_text, created_at
		 FROM command_history WHERE session_id = $1 ORDER BY turn_number ASC LIMIT $2 OFFSET $3`,
		sessionID, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("querying history: %w", err)
	}
	defer rows.Close()

	commands := make([]models.CommandEntry, 0)
	for rows.Next() {
		var cmd models.CommandEntry
		if err := rows.Scan(&cmd.ID, &cmd.SessionID, &cmd.TurnNumber, &cmd.RawInput,
			&cmd.ParsedVerb, &cmd.ParsedTarget, &cmd.RoomID, &cmd.ResponseText, &cmd.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning history row: %w", err)
		}
		commands = append(commands, cmd)
	}

	return &models.HistoryResponse{Commands: commands, Total: total}, nil
}

// --- Internal helpers ---

func (s *GameService) loadWorldState(ctx context.Context, sessionID uuid.UUID) (*engine.WorldState, *engine.Engine, error) {
	var session models.GameSession
	var storyID *uuid.UUID
	err := s.db.QueryRowContext(ctx,
		`SELECT id, current_room_id, turn_number, status, story_id FROM game_sessions WHERE id = $1`,
		sessionID,
	).Scan(&session.ID, &session.CurrentRoomID, &session.TurnNumber, &session.Status, &storyID)
	if err == sql.ErrNoRows {
		return nil, nil, models.NewNotFoundError("game session")
	}
	if err != nil {
		return nil, nil, fmt.Errorf("loading session: %w", err)
	}

	// Get the correct engine for this session's story
	var eng *engine.Engine
	if storyID != nil {
		var loadErr error
		eng, _, loadErr = s.getEngine(ctx, *storyID)
		if loadErr != nil {
			return nil, nil, fmt.Errorf("loading engine for story %s: %w", storyID, loadErr)
		}
	} else {
		eng = s.fallbackEngine
	}

	state := &engine.WorldState{
		SessionID:   sessionID.String(),
		CurrentRoom: session.CurrentRoomID,
		TurnNumber:  session.TurnNumber,
		Status:      session.Status,
		Inventory:   make(map[string]bool),
		Variables:   make(map[string]engine.Variable),
		RoomStates:  make(map[string]*engine.RoomState),
		NpcStates:   make(map[string]*engine.NpcState),
	}

	for roomID := range eng.World.Rooms {
		state.RoomStates[roomID] = &engine.RoomState{
			AddedItems:         make(map[string]bool),
			RemovedItems:       make(map[string]bool),
			BlockedConnections: make(map[string]bool),
			AddedConnections:   make(map[string]string),
		}
	}

	for npcID, npc := range eng.World.Npcs {
		state.NpcStates[npcID] = &engine.NpcState{CurrentRoom: npc.Room}
	}

	// Load inventory
	rows, err := s.db.QueryContext(ctx,
		`SELECT item_id FROM session_inventory WHERE session_id = $1`, sessionID)
	if err != nil {
		return nil, nil, fmt.Errorf("loading inventory: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var itemID string
		if err := rows.Scan(&itemID); err != nil {
			return nil, nil, fmt.Errorf("scanning inventory: %w", err)
		}
		state.Inventory[itemID] = true
	}

	// Load variables
	varRows, err := s.db.QueryContext(ctx,
		`SELECT var_key, var_type, val_bool, val_int, val_string FROM session_variables WHERE session_id = $1`, sessionID)
	if err != nil {
		return nil, nil, fmt.Errorf("loading variables: %w", err)
	}
	defer varRows.Close()
	for varRows.Next() {
		var key, varType string
		var valBool *bool
		var valInt *int
		var valString *string
		if err := varRows.Scan(&key, &varType, &valBool, &valInt, &valString); err != nil {
			return nil, nil, fmt.Errorf("scanning variable: %w", err)
		}
		v := engine.Variable{Type: varType}
		switch varType {
		case "bool":
			if valBool != nil {
				v.BoolVal = *valBool
			}
		case "int":
			if valInt != nil {
				v.IntVal = *valInt
			}
		case "string":
			if valString != nil {
				v.StrVal = *valString
			}
		}
		state.Variables[key] = v
	}

	s.reconstructRoomStates(state)
	s.reconstructNpcStates(state)

	return state, eng, nil
}

func (s *GameService) reconstructRoomStates(state *engine.WorldState) {
	for key, v := range state.Variables {
		if len(key) < 12 {
			continue
		}
		if key[:11] != "room_state." {
			continue
		}
		rest := key[11:]

		dotIdx := -1
		for i, c := range rest {
			if c == '.' {
				dotIdx = i
				break
			}
		}
		if dotIdx < 0 {
			continue
		}
		roomID := rest[:dotIdx]
		remainder := rest[dotIdx+1:]

		rs := state.RoomStates[roomID]
		if rs == nil {
			continue
		}

		if strings.HasPrefix(remainder, "removed_item.") {
			itemID := remainder[len("removed_item."):]
			if v.BoolVal {
				rs.RemovedItems[itemID] = true
			}
		} else if strings.HasPrefix(remainder, "added_item.") {
			itemID := remainder[len("added_item."):]
			if v.BoolVal {
				rs.AddedItems[itemID] = true
			}
		} else if strings.HasPrefix(remainder, "blocked.") {
			dir := remainder[len("blocked."):]
			if v.BoolVal {
				rs.BlockedConnections[dir] = true
			}
		} else if strings.HasPrefix(remainder, "added_conn.") {
			dir := remainder[len("added_conn."):]
			rs.AddedConnections[dir] = v.StrVal
		}
	}
}

func (s *GameService) reconstructNpcStates(state *engine.WorldState) {
	for key, v := range state.Variables {
		if !strings.HasPrefix(key, "npc_room.") {
			continue
		}
		npcID := key[len("npc_room."):]
		if ns, ok := state.NpcStates[npcID]; ok {
			ns.CurrentRoom = v.StrVal
		}
	}
}

func (s *GameService) syncInventory(ctx context.Context, tx *sql.Tx, sessionID uuid.UUID, state *engine.WorldState) error {
	_, err := tx.ExecContext(ctx, `DELETE FROM session_inventory WHERE session_id = $1`, sessionID)
	if err != nil {
		return err
	}

	for itemID := range state.Inventory {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO session_inventory (session_id, item_id, acquired_at_turn) VALUES ($1, $2, $3)`,
			sessionID, itemID, state.TurnNumber,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *GameService) syncVariables(ctx context.Context, tx *sql.Tx, sessionID uuid.UUID, state *engine.WorldState) error {
	_, err := tx.ExecContext(ctx, `DELETE FROM session_variables WHERE session_id = $1`, sessionID)
	if err != nil {
		return err
	}

	for key, v := range state.Variables {
		if len(key) >= 11 && key[:11] == "room_state." {
			continue
		}
		if strings.HasPrefix(key, "npc_room.") {
			continue
		}
		var valBool *bool
		var valInt *int
		var valString *string
		switch v.Type {
		case "bool":
			valBool = &v.BoolVal
		case "int":
			valInt = &v.IntVal
		case "string":
			valString = &v.StrVal
		}
		_, err := tx.ExecContext(ctx,
			`INSERT INTO session_variables (session_id, var_key, var_type, val_bool, val_int, val_string)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			sessionID, key, v.Type, valBool, valInt, valString,
		)
		if err != nil {
			return err
		}
	}

	for roomID, rs := range state.RoomStates {
		for itemID := range rs.RemovedItems {
			key := fmt.Sprintf("room_state.%s.removed_item.%s", roomID, itemID)
			b := true
			_, err := tx.ExecContext(ctx,
				`INSERT INTO session_variables (session_id, var_key, var_type, val_bool) VALUES ($1, $2, 'bool', $3)`,
				sessionID, key, &b,
			)
			if err != nil {
				return err
			}
		}
		for itemID := range rs.AddedItems {
			key := fmt.Sprintf("room_state.%s.added_item.%s", roomID, itemID)
			b := true
			_, err := tx.ExecContext(ctx,
				`INSERT INTO session_variables (session_id, var_key, var_type, val_bool) VALUES ($1, $2, 'bool', $3)`,
				sessionID, key, &b,
			)
			if err != nil {
				return err
			}
		}
		for dir := range rs.BlockedConnections {
			key := fmt.Sprintf("room_state.%s.blocked.%s", roomID, dir)
			b := true
			_, err := tx.ExecContext(ctx,
				`INSERT INTO session_variables (session_id, var_key, var_type, val_bool) VALUES ($1, $2, 'bool', $3)`,
				sessionID, key, &b,
			)
			if err != nil {
				return err
			}
		}
		for dir, target := range rs.AddedConnections {
			key := fmt.Sprintf("room_state.%s.added_conn.%s", roomID, dir)
			_, err := tx.ExecContext(ctx,
				`INSERT INTO session_variables (session_id, var_key, var_type, val_string) VALUES ($1, $2, 'string', $3)`,
				sessionID, key, target,
			)
			if err != nil {
				return err
			}
		}
	}

	// Persist NPC positions
	for npcID, ns := range state.NpcStates {
		key := fmt.Sprintf("npc_room.%s", npcID)
		_, err := tx.ExecContext(ctx,
			`INSERT INTO session_variables (session_id, var_key, var_type, val_string) VALUES ($1, $2, 'string', $3)`,
			sessionID, key, ns.CurrentRoom,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func toItemInfoResponses(items []engine.ItemInfo) []models.ItemInfoResponse {
	result := make([]models.ItemInfoResponse, len(items))
	for i, item := range items {
		result[i] = models.ItemInfoResponse{
			ID:          item.ID,
			Name:        item.Name,
			Description: item.Description,
		}
	}
	return result
}
