package services

import (
	"context"
	"fmt"

	"temple-adventure/engine"
	"temple-adventure/models"

	"github.com/google/uuid"
)

type StoryService struct {
	repo  *StoryRepository
	cache *engine.EngineCache
}

func NewStoryService(repo *StoryRepository, cache *engine.EngineCache) *StoryService {
	return &StoryService{repo: repo, cache: cache}
}

func (s *StoryService) Create(ctx context.Context, req models.CreateStoryRequest) (*models.Story, error) {
	if req.Name == "" {
		return nil, &models.APIError{StatusCode: 400, Message: "name is required"}
	}
	if req.Slug == "" {
		return nil, &models.APIError{StatusCode: 400, Message: "slug is required"}
	}
	if req.StartRoom == "" {
		req.StartRoom = "entrance"
	}
	if req.Author == "" {
		req.Author = "Anonymous"
	}
	return s.repo.Create(ctx, req)
}

func (s *StoryService) GetByID(ctx context.Context, id uuid.UUID) (*models.StoryResponse, error) {
	story, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	world, err := s.repo.LoadWorldDefinition(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("loading world: %w", err)
	}

	return &models.StoryResponse{
		Story:   *story,
		Rooms:   world.Rooms,
		Items:   world.Items,
		Puzzles: world.Puzzles,
	}, nil
}

func (s *StoryService) List(ctx context.Context, publishedOnly bool) (*models.StoryListResponse, error) {
	stories, err := s.repo.List(ctx, publishedOnly)
	if err != nil {
		return nil, err
	}
	return &models.StoryListResponse{Stories: stories}, nil
}

func (s *StoryService) Update(ctx context.Context, id uuid.UUID, req models.UpdateStoryRequest) (*models.Story, error) {
	story, err := s.repo.Update(ctx, id, req)
	if err != nil {
		return nil, err
	}
	s.cache.Invalidate(id)
	return story, nil
}

func (s *StoryService) Delete(ctx context.Context, id uuid.UUID) error {
	err := s.repo.Delete(ctx, id)
	if err != nil {
		return err
	}
	s.cache.Invalidate(id)
	return nil
}

func (s *StoryService) Validate(ctx context.Context, id uuid.UUID) (*models.ValidateResponse, error) {
	story, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	world, err := s.repo.LoadWorldDefinition(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("loading world: %w", err)
	}

	var errors []models.ValidationError

	if len(world.Rooms) == 0 {
		errors = append(errors, models.ValidationError{Field: "rooms", Message: "no rooms defined"})
	}

	if _, ok := world.Rooms[story.StartRoom]; !ok {
		errors = append(errors, models.ValidationError{
			Field:   "start_room",
			Message: fmt.Sprintf("start room %q not found", story.StartRoom),
		})
	}

	// Use engine validation for cross-references
	if err := world.ValidateWithStartRoom(story.StartRoom); err != nil {
		errors = append(errors, models.ValidationError{Field: "world", Message: err.Error()})
	}

	return &models.ValidateResponse{
		Valid:  len(errors) == 0,
		Errors: errors,
	}, nil
}

func (s *StoryService) Publish(ctx context.Context, id uuid.UUID) error {
	validation, err := s.Validate(ctx, id)
	if err != nil {
		return err
	}
	if !validation.Valid {
		return &models.APIError{
			StatusCode: 422,
			Message:    fmt.Sprintf("story has %d validation error(s)", len(validation.Errors)),
		}
	}
	return s.repo.SetPublished(ctx, id, true)
}

func (s *StoryService) Unpublish(ctx context.Context, id uuid.UUID) error {
	return s.repo.SetPublished(ctx, id, false)
}

// --- Content CRUD ---

func (s *StoryService) UpsertRoom(ctx context.Context, storyID uuid.UUID, roomID string, req models.UpsertRoomRequest) error {
	if err := s.repo.UpsertRoom(ctx, storyID, roomID, req); err != nil {
		return err
	}
	s.cache.Invalidate(storyID)
	return nil
}

func (s *StoryService) DeleteRoom(ctx context.Context, storyID uuid.UUID, roomID string) error {
	if err := s.repo.DeleteRoom(ctx, storyID, roomID); err != nil {
		return err
	}
	s.cache.Invalidate(storyID)
	return nil
}

func (s *StoryService) UpsertItem(ctx context.Context, storyID uuid.UUID, itemID string, req models.UpsertItemRequest) error {
	if err := s.repo.UpsertItem(ctx, storyID, itemID, req); err != nil {
		return err
	}
	s.cache.Invalidate(storyID)
	return nil
}

func (s *StoryService) DeleteItem(ctx context.Context, storyID uuid.UUID, itemID string) error {
	if err := s.repo.DeleteItem(ctx, storyID, itemID); err != nil {
		return err
	}
	s.cache.Invalidate(storyID)
	return nil
}

func (s *StoryService) UpsertPuzzle(ctx context.Context, storyID uuid.UUID, puzzleID string, req models.UpsertPuzzleRequest) error {
	if err := s.repo.UpsertPuzzle(ctx, storyID, puzzleID, req); err != nil {
		return err
	}
	s.cache.Invalidate(storyID)
	return nil
}

func (s *StoryService) DeletePuzzle(ctx context.Context, storyID uuid.UUID, puzzleID string) error {
	if err := s.repo.DeletePuzzle(ctx, storyID, puzzleID); err != nil {
		return err
	}
	s.cache.Invalidate(storyID)
	return nil
}
