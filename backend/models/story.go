package models

import (
	"time"

	"temple-adventure/engine"

	"github.com/google/uuid"
)

// --- Database models ---

type Story struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	Author      string    `json:"author"`
	StartRoom   string    `json:"start_room"`
	IsPublished bool      `json:"is_published"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type StorySummary struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	Author      string    `json:"author"`
	IsPublished bool      `json:"is_published"`
	AvgRating   float64   `json:"avg_rating"`
	RatingCount int       `json:"rating_count"`
}

// --- API request types ---

type CreateStoryRequest struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Author      string `json:"author"`
	StartRoom   string `json:"start_room"`
}

type UpdateStoryRequest struct {
	Name        *string `json:"name,omitempty"`
	Slug        *string `json:"slug,omitempty"`
	Description *string `json:"description,omitempty"`
	Author      *string `json:"author,omitempty"`
	StartRoom   *string `json:"start_room,omitempty"`
}

type CreateGameRequest struct {
	StoryID uuid.UUID `json:"story_id"`
}

type UpsertRoomRequest struct {
	Name                    string                    `json:"name"`
	Description             string                    `json:"description"`
	Connections             map[string]string          `json:"connections"`
	Items                   []string                   `json:"items"`
	Puzzles                 []string                   `json:"puzzles"`
	ConditionalDescriptions []engine.ConditionalText   `json:"conditional_descriptions"`
	Hints                   []engine.ConditionalHint   `json:"hints"`
}

type UpsertItemRequest struct {
	Name                    string                    `json:"name"`
	Aliases                 []string                   `json:"aliases"`
	Description             string                    `json:"description"`
	Portable                bool                      `json:"portable"`
	Interactions            []engine.Interaction       `json:"interactions"`
	ConditionalDescriptions []engine.ConditionalText   `json:"conditional_descriptions"`
}

type UpsertPuzzleRequest struct {
	Name           string              `json:"name"`
	Description    string              `json:"description"`
	Steps          []engine.PuzzleStep `json:"steps"`
	TimedWindow    *engine.TimedWindow `json:"timed_window"`
	FailureEffects []engine.Effect     `json:"failure_effects"`
	FailureText    string              `json:"failure_text"`
	CompletionText string              `json:"completion_text"`
}

type UpsertNpcRequest struct {
	Name                    string                    `json:"name"`
	Description             string                    `json:"description"`
	Aliases                 []string                  `json:"aliases"`
	Room                    string                    `json:"room"`
	Dialogue                []engine.DialogueLine     `json:"dialogue"`
	Movement                []engine.NpcMovement      `json:"movement"`
	ConditionalDescriptions []engine.ConditionalText  `json:"conditional_descriptions"`
}

// --- API response types ---

type StoryResponse struct {
	Story   Story                        `json:"story"`
	Rooms   map[string]*engine.RoomDef   `json:"rooms"`
	Items   map[string]*engine.ItemDef   `json:"items"`
	Puzzles map[string]*engine.PuzzleDef `json:"puzzles"`
	Npcs    map[string]*engine.NpcDef    `json:"npcs"`
}

type StoryListResponse struct {
	Stories []StorySummary `json:"stories"`
	Total   int            `json:"total"`
	Limit   int            `json:"limit"`
	Offset  int            `json:"offset"`
}

type RateStoryRequest struct {
	Rating int `json:"rating"`
}

type StoryRating struct {
	ID        uuid.UUID `json:"id"`
	StoryID   uuid.UUID `json:"story_id"`
	SessionID uuid.UUID `json:"session_id"`
	Rating    int       `json:"rating"`
	CreatedAt time.Time `json:"created_at"`
}

type StoryRatingResponse struct {
	AvgRating   float64 `json:"avg_rating"`
	RatingCount int     `json:"rating_count"`
	UserRating  *int    `json:"user_rating,omitempty"`
}

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ValidateResponse struct {
	Valid  bool              `json:"valid"`
	Errors []ValidationError `json:"errors"`
}
