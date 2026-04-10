package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"temple-adventure/engine"
	"temple-adventure/models"

	"github.com/google/uuid"
)

type StoryRepository struct {
	db *sql.DB
}

func NewStoryRepository(db *sql.DB) *StoryRepository {
	return &StoryRepository{db: db}
}

// --- Story CRUD ---

func (r *StoryRepository) Create(ctx context.Context, req models.CreateStoryRequest) (*models.Story, error) {
	var story models.Story
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO stories (name, slug, description, author, start_room)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, name, slug, description, author, start_room, is_published, created_at, updated_at`,
		req.Name, req.Slug, req.Description, req.Author, req.StartRoom,
	).Scan(&story.ID, &story.Name, &story.Slug, &story.Description, &story.Author,
		&story.StartRoom, &story.IsPublished, &story.CreatedAt, &story.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating story: %w", err)
	}
	return &story, nil
}

func (r *StoryRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Story, error) {
	var story models.Story
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, slug, description, author, start_room, is_published, created_at, updated_at
		 FROM stories WHERE id = $1`, id,
	).Scan(&story.ID, &story.Name, &story.Slug, &story.Description, &story.Author,
		&story.StartRoom, &story.IsPublished, &story.CreatedAt, &story.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, models.NewNotFoundError("story")
	}
	if err != nil {
		return nil, fmt.Errorf("getting story: %w", err)
	}
	return &story, nil
}

func (r *StoryRepository) GetBySlug(ctx context.Context, slug string) (*models.Story, error) {
	var story models.Story
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, slug, description, author, start_room, is_published, created_at, updated_at
		 FROM stories WHERE slug = $1`, slug,
	).Scan(&story.ID, &story.Name, &story.Slug, &story.Description, &story.Author,
		&story.StartRoom, &story.IsPublished, &story.CreatedAt, &story.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting story by slug: %w", err)
	}
	return &story, nil
}

func (r *StoryRepository) List(ctx context.Context, publishedOnly bool) ([]models.StorySummary, error) {
	query := `SELECT id, name, slug, description, author, is_published FROM stories`
	if publishedOnly {
		query += ` WHERE is_published = true`
	}
	query += ` ORDER BY name`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("listing stories: %w", err)
	}
	defer rows.Close()

	stories := make([]models.StorySummary, 0)
	for rows.Next() {
		var s models.StorySummary
		if err := rows.Scan(&s.ID, &s.Name, &s.Slug, &s.Description, &s.Author, &s.IsPublished); err != nil {
			return nil, fmt.Errorf("scanning story: %w", err)
		}
		stories = append(stories, s)
	}
	return stories, nil
}

func (r *StoryRepository) Update(ctx context.Context, id uuid.UUID, req models.UpdateStoryRequest) (*models.Story, error) {
	// Build dynamic update
	story, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		story.Name = *req.Name
	}
	if req.Slug != nil {
		story.Slug = *req.Slug
	}
	if req.Description != nil {
		story.Description = *req.Description
	}
	if req.Author != nil {
		story.Author = *req.Author
	}
	if req.StartRoom != nil {
		story.StartRoom = *req.StartRoom
	}

	err = r.db.QueryRowContext(ctx,
		`UPDATE stories SET name = $1, slug = $2, description = $3, author = $4, start_room = $5
		 WHERE id = $6
		 RETURNING id, name, slug, description, author, start_room, is_published, created_at, updated_at`,
		story.Name, story.Slug, story.Description, story.Author, story.StartRoom, id,
	).Scan(&story.ID, &story.Name, &story.Slug, &story.Description, &story.Author,
		&story.StartRoom, &story.IsPublished, &story.CreatedAt, &story.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("updating story: %w", err)
	}
	return story, nil
}

func (r *StoryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM stories WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting story: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return models.NewNotFoundError("story")
	}
	return nil
}

func (r *StoryRepository) SetPublished(ctx context.Context, id uuid.UUID, published bool) error {
	_, err := r.db.ExecContext(ctx, `UPDATE stories SET is_published = $1 WHERE id = $2`, published, id)
	return err
}

// --- Room CRUD ---

func (r *StoryRepository) UpsertRoom(ctx context.Context, storyID uuid.UUID, roomID string, req models.UpsertRoomRequest) error {
	connections, _ := json.Marshal(req.Connections)
	items, _ := json.Marshal(req.Items)
	puzzles, _ := json.Marshal(req.Puzzles)
	condDescs, _ := json.Marshal(req.ConditionalDescriptions)
	hints, _ := json.Marshal(req.Hints)

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO story_rooms (story_id, room_id, name, description, connections, items, puzzles, conditional_descriptions, hints)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 ON CONFLICT (story_id, room_id) DO UPDATE SET
		   name = EXCLUDED.name, description = EXCLUDED.description, connections = EXCLUDED.connections,
		   items = EXCLUDED.items, puzzles = EXCLUDED.puzzles, conditional_descriptions = EXCLUDED.conditional_descriptions,
		   hints = EXCLUDED.hints`,
		storyID, roomID, req.Name, req.Description, connections, items, puzzles, condDescs, hints,
	)
	return err
}

func (r *StoryRepository) DeleteRoom(ctx context.Context, storyID uuid.UUID, roomID string) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM story_rooms WHERE story_id = $1 AND room_id = $2`, storyID, roomID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return models.NewNotFoundError("room")
	}
	return nil
}

// --- Item CRUD ---

func (r *StoryRepository) UpsertItem(ctx context.Context, storyID uuid.UUID, itemID string, req models.UpsertItemRequest) error {
	aliases, _ := json.Marshal(req.Aliases)
	interactions, _ := json.Marshal(req.Interactions)
	condDescs, _ := json.Marshal(req.ConditionalDescriptions)

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO story_items (story_id, item_id, name, aliases, description, portable, interactions, conditional_descriptions)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT (story_id, item_id) DO UPDATE SET
		   name = EXCLUDED.name, aliases = EXCLUDED.aliases, description = EXCLUDED.description,
		   portable = EXCLUDED.portable, interactions = EXCLUDED.interactions,
		   conditional_descriptions = EXCLUDED.conditional_descriptions`,
		storyID, itemID, req.Name, aliases, req.Description, req.Portable, interactions, condDescs,
	)
	return err
}

func (r *StoryRepository) DeleteItem(ctx context.Context, storyID uuid.UUID, itemID string) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM story_items WHERE story_id = $1 AND item_id = $2`, storyID, itemID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return models.NewNotFoundError("item")
	}
	return nil
}

// --- Puzzle CRUD ---

func (r *StoryRepository) UpsertPuzzle(ctx context.Context, storyID uuid.UUID, puzzleID string, req models.UpsertPuzzleRequest) error {
	steps, _ := json.Marshal(req.Steps)
	timedWindow, _ := json.Marshal(req.TimedWindow)
	failureEffects, _ := json.Marshal(req.FailureEffects)

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO story_puzzles (story_id, puzzle_id, name, description, steps, timed_window, failure_effects, failure_text, completion_text)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 ON CONFLICT (story_id, puzzle_id) DO UPDATE SET
		   name = EXCLUDED.name, description = EXCLUDED.description, steps = EXCLUDED.steps,
		   timed_window = EXCLUDED.timed_window, failure_effects = EXCLUDED.failure_effects,
		   failure_text = EXCLUDED.failure_text, completion_text = EXCLUDED.completion_text`,
		storyID, puzzleID, req.Name, req.Description, steps, timedWindow, failureEffects, req.FailureText, req.CompletionText,
	)
	return err
}

func (r *StoryRepository) DeletePuzzle(ctx context.Context, storyID uuid.UUID, puzzleID string) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM story_puzzles WHERE story_id = $1 AND puzzle_id = $2`, storyID, puzzleID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return models.NewNotFoundError("puzzle")
	}
	return nil
}

// --- Load as WorldDefinition ---

func (r *StoryRepository) LoadWorldDefinition(ctx context.Context, storyID uuid.UUID) (*engine.WorldDefinition, error) {
	world := &engine.WorldDefinition{
		Rooms:   make(map[string]*engine.RoomDef),
		Items:   make(map[string]*engine.ItemDef),
		Puzzles: make(map[string]*engine.PuzzleDef),
	}

	// Load rooms
	roomRows, err := r.db.QueryContext(ctx,
		`SELECT room_id, name, description, connections, items, puzzles, conditional_descriptions, hints
		 FROM story_rooms WHERE story_id = $1`, storyID)
	if err != nil {
		return nil, fmt.Errorf("loading rooms: %w", err)
	}
	defer roomRows.Close()

	for roomRows.Next() {
		var roomID, name, desc string
		var connectionsJSON, itemsJSON, puzzlesJSON, condDescsJSON, hintsJSON []byte

		if err := roomRows.Scan(&roomID, &name, &desc, &connectionsJSON, &itemsJSON, &puzzlesJSON, &condDescsJSON, &hintsJSON); err != nil {
			return nil, fmt.Errorf("scanning room: %w", err)
		}

		room := &engine.RoomDef{
			ID:          roomID,
			Name:        name,
			Description: desc,
		}
		json.Unmarshal(connectionsJSON, &room.Connections)
		json.Unmarshal(itemsJSON, &room.Items)
		json.Unmarshal(puzzlesJSON, &room.Puzzles)
		json.Unmarshal(condDescsJSON, &room.ConditionalDescriptions)
		json.Unmarshal(hintsJSON, &room.Hints)

		if room.Connections == nil {
			room.Connections = make(map[string]string)
		}

		world.Rooms[roomID] = room
	}

	// Load items
	itemRows, err := r.db.QueryContext(ctx,
		`SELECT item_id, name, aliases, description, portable, interactions, conditional_descriptions
		 FROM story_items WHERE story_id = $1`, storyID)
	if err != nil {
		return nil, fmt.Errorf("loading items: %w", err)
	}
	defer itemRows.Close()

	for itemRows.Next() {
		var itemID, name, desc string
		var portable bool
		var aliasesJSON, interactionsJSON, condDescsJSON []byte

		if err := itemRows.Scan(&itemID, &name, &aliasesJSON, &desc, &portable, &interactionsJSON, &condDescsJSON); err != nil {
			return nil, fmt.Errorf("scanning item: %w", err)
		}

		item := &engine.ItemDef{
			ID:          itemID,
			Name:        name,
			Description: desc,
			Portable:    portable,
		}
		json.Unmarshal(aliasesJSON, &item.Aliases)
		json.Unmarshal(interactionsJSON, &item.Interactions)
		json.Unmarshal(condDescsJSON, &item.ConditionalDescriptions)

		world.Items[itemID] = item
	}

	// Load puzzles
	puzzleRows, err := r.db.QueryContext(ctx,
		`SELECT puzzle_id, name, description, steps, timed_window, failure_effects, failure_text, completion_text
		 FROM story_puzzles WHERE story_id = $1`, storyID)
	if err != nil {
		return nil, fmt.Errorf("loading puzzles: %w", err)
	}
	defer puzzleRows.Close()

	for puzzleRows.Next() {
		var puzzleID, name, desc, failureText, completionText string
		var stepsJSON, timedWindowJSON, failureEffectsJSON []byte

		if err := puzzleRows.Scan(&puzzleID, &name, &desc, &stepsJSON, &timedWindowJSON, &failureEffectsJSON, &failureText, &completionText); err != nil {
			return nil, fmt.Errorf("scanning puzzle: %w", err)
		}

		puzzle := &engine.PuzzleDef{
			ID:             puzzleID,
			Name:           name,
			Description:    desc,
			FailureText:    failureText,
			CompletionText: completionText,
		}
		json.Unmarshal(stepsJSON, &puzzle.Steps)
		json.Unmarshal(failureEffectsJSON, &puzzle.FailureEffects)
		if timedWindowJSON != nil {
			json.Unmarshal(timedWindowJSON, &puzzle.TimedWindow)
		}

		world.Puzzles[puzzleID] = puzzle
	}

	return world, nil
}
