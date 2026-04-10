package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"temple-adventure/engine"
)

// SeedDefaultStory loads the YAML content from contentDir and seeds it into the
// stories tables if it doesn't already exist. Returns the story ID.
func SeedDefaultStory(ctx context.Context, db *sql.DB, contentDir string) error {
	repo := NewStoryRepository(db)

	// Check if already seeded
	existing, err := repo.GetBySlug(ctx, "temple-of-the-sun")
	if err != nil {
		return fmt.Errorf("checking for existing story: %w", err)
	}
	if existing != nil {
		log.Printf("Default story already seeded (id: %s)", existing.ID)
		// Backfill any game sessions missing story_id
		_, err = db.ExecContext(ctx,
			`UPDATE game_sessions SET story_id = $1 WHERE story_id IS NULL`, existing.ID)
		return err
	}

	// Load from YAML
	world, err := engine.LoadWorldDefinition(contentDir)
	if err != nil {
		return fmt.Errorf("loading world definition for seeding: %w", err)
	}

	// Create story
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning seed transaction: %w", err)
	}
	defer tx.Rollback()

	var storyID string
	err = tx.QueryRowContext(ctx,
		`INSERT INTO stories (name, slug, description, author, start_room, is_published)
		 VALUES ($1, $2, $3, $4, $5, true)
		 RETURNING id`,
		"Temple of the Sun",
		"temple-of-the-sun",
		"An adventure in a lost Inca temple. Solve ancient puzzles, navigate deadly traps, and claim the legendary Sun Crown.",
		"Temple Adventure Team",
		"entrance",
	).Scan(&storyID)
	if err != nil {
		return fmt.Errorf("inserting story: %w", err)
	}

	// Insert rooms
	for roomID, room := range world.Rooms {
		connections, _ := json.Marshal(room.Connections)
		items, _ := json.Marshal(room.Items)
		puzzles, _ := json.Marshal(room.Puzzles)
		condDescs, _ := json.Marshal(room.ConditionalDescriptions)
		hints, _ := json.Marshal(room.Hints)

		_, err = tx.ExecContext(ctx,
			`INSERT INTO story_rooms (story_id, room_id, name, description, connections, items, puzzles, conditional_descriptions, hints)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			storyID, roomID, room.Name, room.Description, connections, items, puzzles, condDescs, hints,
		)
		if err != nil {
			return fmt.Errorf("inserting room %s: %w", roomID, err)
		}
	}

	// Insert items
	for itemID, item := range world.Items {
		aliases, _ := json.Marshal(item.Aliases)
		interactions, _ := json.Marshal(item.Interactions)
		condDescs, _ := json.Marshal(item.ConditionalDescriptions)

		_, err = tx.ExecContext(ctx,
			`INSERT INTO story_items (story_id, item_id, name, aliases, description, portable, interactions, conditional_descriptions)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			storyID, itemID, item.Name, aliases, item.Description, item.Portable, interactions, condDescs,
		)
		if err != nil {
			return fmt.Errorf("inserting item %s: %w", itemID, err)
		}
	}

	// Insert puzzles
	for puzzleID, puzzle := range world.Puzzles {
		steps, _ := json.Marshal(puzzle.Steps)
		timedWindow, _ := json.Marshal(puzzle.TimedWindow)
		failureEffects, _ := json.Marshal(puzzle.FailureEffects)

		_, err = tx.ExecContext(ctx,
			`INSERT INTO story_puzzles (story_id, puzzle_id, name, description, steps, timed_window, failure_effects, failure_text, completion_text)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			storyID, puzzleID, puzzle.Name, puzzle.Description, steps, timedWindow, failureEffects, puzzle.FailureText, puzzle.CompletionText,
		)
		if err != nil {
			return fmt.Errorf("inserting puzzle %s: %w", puzzleID, err)
		}
	}

	// Backfill existing sessions
	_, err = tx.ExecContext(ctx,
		`UPDATE game_sessions SET story_id = $1 WHERE story_id IS NULL`, storyID)
	if err != nil {
		return fmt.Errorf("backfilling sessions: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing seed: %w", err)
	}

	log.Printf("Default story seeded (id: %s): %d rooms, %d items, %d puzzles",
		storyID, len(world.Rooms), len(world.Items), len(world.Puzzles))
	return nil
}
