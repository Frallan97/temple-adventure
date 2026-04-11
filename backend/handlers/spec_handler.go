package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"

	"temple-adventure/engine"
	"temple-adventure/services"
	"temple-adventure/storygen"

	chiMiddleware "github.com/go-chi/chi/v5/middleware"
)

type SpecHandler struct {
	repo  *services.StoryRepository
	db    *sql.DB
	debug bool
}

func NewSpecHandler(repo *services.StoryRepository, db *sql.DB, debug bool) *SpecHandler {
	return &SpecHandler{repo: repo, db: db, debug: debug}
}

// CreateFromSpec handles POST /api/v1/stories/from-spec
func (h *SpecHandler) CreateFromSpec(w http.ResponseWriter, r *http.Request) {
	var spec storygen.StorySpec
	if err := DecodeJSON(r, &spec); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

	// Validate spec
	if errs := storygen.ValidateSpec(&spec); len(errs) > 0 {
		WriteJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{
			"error":             "Story spec validation failed",
			"validation_errors": errs,
		})
		return
	}

	// Expand to WorldDefinition
	world, err := storygen.Expand(&spec)
	if err != nil {
		WriteError(w, http.StatusUnprocessableEntity, "Expansion failed: "+err.Error())
		return
	}

	// Deep validate
	if errs := storygen.ValidateWorldDeep(world, spec.StartRoom); len(errs) > 0 {
		WriteJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{
			"error":             "Expanded world validation failed",
			"validation_errors": errs,
		})
		return
	}

	// Save to database
	ctx := r.Context()
	result, err := h.saveStory(ctx, &spec, world)
	if err != nil {
		reqID := chiMiddleware.GetReqID(ctx)
		log.Printf("[%s] Error saving story from spec: %v", reqID, err)
		if h.debug {
			WriteError(w, http.StatusInternalServerError, err.Error())
		} else {
			WriteError(w, http.StatusInternalServerError, "Failed to save story")
		}
		return
	}

	WriteJSON(w, http.StatusCreated, result)
}

func (h *SpecHandler) saveStory(ctx context.Context, spec *storygen.StorySpec, world *engine.WorldDefinition) (map[string]interface{}, error) {
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	slug := spec.Slug
	if slug == "" {
		slug = slugify(spec.Title)
	}
	slug, err = h.ensureUniqueSlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("ensuring unique slug: %w", err)
	}

	author := spec.Author
	if author == "" {
		author = "Anonymous"
	}

	var storyID string
	err = tx.QueryRowContext(ctx,
		`INSERT INTO stories (name, slug, description, author, start_room, is_published)
		 VALUES ($1, $2, $3, $4, $5, true)
		 RETURNING id`,
		spec.Title, slug, spec.Description, author, spec.StartRoom,
	).Scan(&storyID)
	if err != nil {
		return nil, fmt.Errorf("inserting story: %w", err)
	}

	if err := h.repo.SaveWorldDefinition(ctx, tx, storyID, world); err != nil {
		return nil, fmt.Errorf("saving world: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("committing: %w", err)
	}

	return map[string]interface{}{
		"story": map[string]interface{}{
			"id":          storyID,
			"name":        spec.Title,
			"slug":        slug,
			"description": spec.Description,
			"author":      author,
			"start_room":  spec.StartRoom,
		},
		"rooms":   world.Rooms,
		"items":   world.Items,
		"puzzles": world.Puzzles,
		"npcs":    world.Npcs,
	}, nil
}

func (h *SpecHandler) ensureUniqueSlug(ctx context.Context, slug string) (string, error) {
	existing, err := h.repo.GetBySlug(ctx, slug)
	if err != nil {
		return "", err
	}
	if existing == nil {
		return slug, nil
	}
	for i := 2; i < 100; i++ {
		candidate := fmt.Sprintf("%s-%d", slug, i)
		existing, err = h.repo.GetBySlug(ctx, candidate)
		if err != nil {
			return "", err
		}
		if existing == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("could not find unique slug for %q", slug)
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "-")
	var b strings.Builder
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			b.WriteRune(c)
		}
	}
	return b.String()
}
