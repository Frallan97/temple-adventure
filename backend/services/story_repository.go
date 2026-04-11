package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

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
		if strings.Contains(err.Error(), "stories_name_unique") {
			return nil, models.NewConflictError("a story with this name already exists")
		}
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

// --- NPC CRUD ---

func (r *StoryRepository) UpsertNpc(ctx context.Context, storyID uuid.UUID, npcID string, req models.UpsertNpcRequest) error {
	aliases, _ := json.Marshal(req.Aliases)
	dialogue, _ := json.Marshal(req.Dialogue)
	movement, _ := json.Marshal(req.Movement)
	condDescs, _ := json.Marshal(req.ConditionalDescriptions)

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO story_npcs (story_id, npc_id, name, description, aliases, room, dialogue, movement, conditional_descriptions)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 ON CONFLICT (story_id, npc_id) DO UPDATE SET
		   name = EXCLUDED.name, description = EXCLUDED.description, aliases = EXCLUDED.aliases,
		   room = EXCLUDED.room, dialogue = EXCLUDED.dialogue, movement = EXCLUDED.movement,
		   conditional_descriptions = EXCLUDED.conditional_descriptions`,
		storyID, npcID, req.Name, req.Description, aliases, req.Room, dialogue, movement, condDescs,
	)
	return err
}

func (r *StoryRepository) DeleteNpc(ctx context.Context, storyID uuid.UUID, npcID string) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM story_npcs WHERE story_id = $1 AND npc_id = $2`, storyID, npcID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return models.NewNotFoundError("npc")
	}
	return nil
}

// --- Load as WorldDefinition ---

func (r *StoryRepository) LoadWorldDefinition(ctx context.Context, storyID uuid.UUID) (*engine.WorldDefinition, error) {
	world := &engine.WorldDefinition{
		Rooms:   make(map[string]*engine.RoomDef),
		Items:   make(map[string]*engine.ItemDef),
		Puzzles: make(map[string]*engine.PuzzleDef),
		Npcs:    make(map[string]*engine.NpcDef),
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

	// Load NPCs
	npcRows, err := r.db.QueryContext(ctx,
		`SELECT npc_id, name, description, aliases, room, dialogue, movement, conditional_descriptions
		 FROM story_npcs WHERE story_id = $1`, storyID)
	if err != nil {
		return nil, fmt.Errorf("loading npcs: %w", err)
	}
	defer npcRows.Close()

	for npcRows.Next() {
		var npcID, name, desc, room string
		var aliasesJSON, dialogueJSON, movementJSON, condDescsJSON []byte

		if err := npcRows.Scan(&npcID, &name, &desc, &aliasesJSON, &room, &dialogueJSON, &movementJSON, &condDescsJSON); err != nil {
			return nil, fmt.Errorf("scanning npc: %w", err)
		}

		npc := &engine.NpcDef{
			ID:          npcID,
			Name:        name,
			Description: desc,
			Room:        room,
		}
		json.Unmarshal(aliasesJSON, &npc.Aliases)
		json.Unmarshal(dialogueJSON, &npc.Dialogue)
		json.Unmarshal(movementJSON, &npc.Movement)
		json.Unmarshal(condDescsJSON, &npc.ConditionalDescriptions)

		world.Npcs[npcID] = npc
	}

	return world, nil
}

// SaveWorldDefinition inserts all rooms, items, and puzzles for a story
// within an existing transaction.
func (r *StoryRepository) SaveWorldDefinition(ctx context.Context, tx *sql.Tx, storyID string, world *engine.WorldDefinition) error {
	for roomID, room := range world.Rooms {
		connections, _ := json.Marshal(room.Connections)
		items, _ := json.Marshal(room.Items)
		puzzles, _ := json.Marshal(room.Puzzles)
		condDescs, _ := json.Marshal(room.ConditionalDescriptions)
		hints, _ := json.Marshal(room.Hints)

		_, err := tx.ExecContext(ctx,
			`INSERT INTO story_rooms (story_id, room_id, name, description, connections, items, puzzles, conditional_descriptions, hints)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			 ON CONFLICT (story_id, room_id) DO UPDATE SET
			   name = EXCLUDED.name, description = EXCLUDED.description,
			   connections = EXCLUDED.connections, items = EXCLUDED.items,
			   puzzles = EXCLUDED.puzzles, conditional_descriptions = EXCLUDED.conditional_descriptions,
			   hints = EXCLUDED.hints`,
			storyID, roomID, room.Name, room.Description, connections, items, puzzles, condDescs, hints,
		)
		if err != nil {
			return fmt.Errorf("inserting room %s: %w", roomID, err)
		}
	}

	for itemID, item := range world.Items {
		aliases, _ := json.Marshal(item.Aliases)
		interactions, _ := json.Marshal(item.Interactions)
		condDescs, _ := json.Marshal(item.ConditionalDescriptions)

		_, err := tx.ExecContext(ctx,
			`INSERT INTO story_items (story_id, item_id, name, aliases, description, portable, interactions, conditional_descriptions)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			 ON CONFLICT (story_id, item_id) DO UPDATE SET
			   name = EXCLUDED.name, aliases = EXCLUDED.aliases,
			   description = EXCLUDED.description, portable = EXCLUDED.portable,
			   interactions = EXCLUDED.interactions, conditional_descriptions = EXCLUDED.conditional_descriptions`,
			storyID, itemID, item.Name, aliases, item.Description, item.Portable, interactions, condDescs,
		)
		if err != nil {
			return fmt.Errorf("inserting item %s: %w", itemID, err)
		}
	}

	for puzzleID, puzzle := range world.Puzzles {
		steps, _ := json.Marshal(puzzle.Steps)
		timedWindow, _ := json.Marshal(puzzle.TimedWindow)
		failureEffects, _ := json.Marshal(puzzle.FailureEffects)

		_, err := tx.ExecContext(ctx,
			`INSERT INTO story_puzzles (story_id, puzzle_id, name, description, steps, timed_window, failure_effects, failure_text, completion_text)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			 ON CONFLICT (story_id, puzzle_id) DO UPDATE SET
			   name = EXCLUDED.name, description = EXCLUDED.description,
			   steps = EXCLUDED.steps, timed_window = EXCLUDED.timed_window,
			   failure_effects = EXCLUDED.failure_effects, failure_text = EXCLUDED.failure_text,
			   completion_text = EXCLUDED.completion_text`,
			storyID, puzzleID, puzzle.Name, puzzle.Description, steps, timedWindow, failureEffects, puzzle.FailureText, puzzle.CompletionText,
		)
		if err != nil {
			return fmt.Errorf("inserting puzzle %s: %w", puzzleID, err)
		}
	}

	for npcID, npc := range world.Npcs {
		aliases, _ := json.Marshal(npc.Aliases)
		dialogue, _ := json.Marshal(npc.Dialogue)
		movement, _ := json.Marshal(npc.Movement)
		condDescs, _ := json.Marshal(npc.ConditionalDescriptions)

		_, err := tx.ExecContext(ctx,
			`INSERT INTO story_npcs (story_id, npc_id, name, description, aliases, room, dialogue, movement, conditional_descriptions)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			 ON CONFLICT (story_id, npc_id) DO UPDATE SET
			   name = EXCLUDED.name, description = EXCLUDED.description,
			   aliases = EXCLUDED.aliases, room = EXCLUDED.room,
			   dialogue = EXCLUDED.dialogue, movement = EXCLUDED.movement,
			   conditional_descriptions = EXCLUDED.conditional_descriptions`,
			storyID, npcID, npc.Name, npc.Description, aliases, npc.Room, dialogue, movement, condDescs,
		)
		if err != nil {
			return fmt.Errorf("inserting npc %s: %w", npcID, err)
		}
	}

	return nil
}
