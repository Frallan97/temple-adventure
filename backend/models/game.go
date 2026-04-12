package models

import (
	"time"

	"github.com/google/uuid"
)

type GameSession struct {
	ID            uuid.UUID `json:"id"`
	CurrentRoomID string    `json:"current_room_id"`
	TurnNumber    int       `json:"turn_number"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type SessionInventory struct {
	ID             uuid.UUID `json:"id"`
	SessionID      uuid.UUID `json:"session_id"`
	ItemID         string    `json:"item_id"`
	AcquiredAtTurn int       `json:"acquired_at_turn"`
}

type SessionVariable struct {
	ID        uuid.UUID `json:"id"`
	SessionID uuid.UUID `json:"session_id"`
	VarKey    string    `json:"var_key"`
	VarType   string    `json:"var_type"`
	ValBool   *bool     `json:"val_bool,omitempty"`
	ValInt    *int      `json:"val_int,omitempty"`
	ValString *string   `json:"val_string,omitempty"`
}

type CommandEntry struct {
	ID           uuid.UUID `json:"id"`
	SessionID    uuid.UUID `json:"session_id"`
	TurnNumber   int       `json:"turn_number"`
	RawInput     string    `json:"raw_input"`
	ParsedVerb   string    `json:"parsed_verb"`
	ParsedTarget string    `json:"parsed_target"`
	RoomID       string    `json:"room_id"`
	ResponseText string    `json:"response_text"`
	CreatedAt    time.Time `json:"created_at"`
}

// API request/response types

type CommandRequest struct {
	Input string `json:"input"`
}

type CreateGameResponse struct {
	ID          uuid.UUID          `json:"id"`
	StoryID     uuid.UUID          `json:"story_id"`
	RoomName    string             `json:"room_name"`
	Description string             `json:"description"`
	TurnNumber  int                `json:"turn_number"`
	Inventory   []ItemInfoResponse `json:"inventory"`
}

type CommandResponse struct {
	Text        string             `json:"text"`
	RoomName    string             `json:"room_name"`
	RoomChanged bool               `json:"room_changed"`
	TurnNumber  int                `json:"turn_number"`
	GameOver    bool               `json:"game_over"`
	GameStatus  string             `json:"game_status"`
	Inventory   []ItemInfoResponse `json:"inventory"`
	Choices     []ChoiceResponse   `json:"choices,omitempty"`
	EndingID    string             `json:"ending_id,omitempty"`
	EndingTitle string             `json:"ending_title,omitempty"`
}

type ChoiceResponse struct {
	Index int    `json:"index"`
	Text  string `json:"text"`
}

type GameStateResponse struct {
	ID          uuid.UUID          `json:"id"`
	RoomName    string             `json:"room_name"`
	Description string             `json:"description"`
	TurnNumber  int                `json:"turn_number"`
	Status      string             `json:"status"`
	Inventory   []ItemInfoResponse `json:"inventory"`
}

type HistoryResponse struct {
	Commands []CommandEntry `json:"commands"`
	Total    int            `json:"total"`
}

type GameLogSummary struct {
	ID         uuid.UUID `json:"id"`
	StoryID    uuid.UUID `json:"story_id"`
	StoryName  string    `json:"story_name"`
	Status     string    `json:"status"`
	TurnNumber int       `json:"turn_number"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type GameLogsResponse struct {
	Games []GameLogSummary `json:"games"`
}

type ItemInfoResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}
