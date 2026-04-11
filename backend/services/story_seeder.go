package services

import (
	"context"
	"database/sql"
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

	// Insert world content using shared helper
	if err := repo.SaveWorldDefinition(ctx, tx, storyID, world); err != nil {
		return fmt.Errorf("saving world definition: %w", err)
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
