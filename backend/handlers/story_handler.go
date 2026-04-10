package handlers

import (
	"log"
	"net/http"

	"temple-adventure/models"
	"temple-adventure/services"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

type StoryHandler struct {
	storyService *services.StoryService
	debug        bool
}

func NewStoryHandler(storyService *services.StoryService, debug bool) *StoryHandler {
	return &StoryHandler{storyService: storyService, debug: debug}
}

func (h *StoryHandler) handleError(w http.ResponseWriter, r *http.Request, err error, fallbackMsg string) {
	if apiErr, ok := err.(*models.APIError); ok {
		WriteError(w, apiErr.StatusCode, apiErr.Message)
		return
	}

	reqID := chiMiddleware.GetReqID(r.Context())
	log.Printf("[ERROR] [req:%s] %s %s — %v", reqID, r.Method, r.URL.Path, err)

	if h.debug {
		WriteError(w, http.StatusInternalServerError, err.Error())
	} else {
		WriteError(w, http.StatusInternalServerError, fallbackMsg)
	}
}

func (h *StoryHandler) ListStories(w http.ResponseWriter, r *http.Request) {
	// Editor requests get all stories, player requests only published
	publishedOnly := r.URL.Query().Get("all") != "true"

	resp, err := h.storyService.List(r.Context(), publishedOnly)
	if err != nil {
		h.handleError(w, r, err, "Failed to list stories")
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *StoryHandler) CreateStory(w http.ResponseWriter, r *http.Request) {
	var req models.CreateStoryRequest
	if err := DecodeJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	story, err := h.storyService.Create(r.Context(), req)
	if err != nil {
		h.handleError(w, r, err, "Failed to create story")
		return
	}
	WriteJSON(w, http.StatusCreated, story)
}

func (h *StoryHandler) GetStory(w http.ResponseWriter, r *http.Request) {
	storyID, err := uuid.Parse(chi.URLParam(r, "storyId"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid story ID")
		return
	}

	resp, err := h.storyService.GetByID(r.Context(), storyID)
	if err != nil {
		h.handleError(w, r, err, "Failed to get story")
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *StoryHandler) UpdateStory(w http.ResponseWriter, r *http.Request) {
	storyID, err := uuid.Parse(chi.URLParam(r, "storyId"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid story ID")
		return
	}

	var req models.UpdateStoryRequest
	if err := DecodeJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	story, err := h.storyService.Update(r.Context(), storyID, req)
	if err != nil {
		h.handleError(w, r, err, "Failed to update story")
		return
	}
	WriteJSON(w, http.StatusOK, story)
}

func (h *StoryHandler) DeleteStory(w http.ResponseWriter, r *http.Request) {
	storyID, err := uuid.Parse(chi.URLParam(r, "storyId"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid story ID")
		return
	}

	if err := h.storyService.Delete(r.Context(), storyID); err != nil {
		h.handleError(w, r, err, "Failed to delete story")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *StoryHandler) ValidateStory(w http.ResponseWriter, r *http.Request) {
	storyID, err := uuid.Parse(chi.URLParam(r, "storyId"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid story ID")
		return
	}

	resp, err := h.storyService.Validate(r.Context(), storyID)
	if err != nil {
		h.handleError(w, r, err, "Failed to validate story")
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *StoryHandler) PublishStory(w http.ResponseWriter, r *http.Request) {
	storyID, err := uuid.Parse(chi.URLParam(r, "storyId"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid story ID")
		return
	}

	if err := h.storyService.Publish(r.Context(), storyID); err != nil {
		h.handleError(w, r, err, "Failed to publish story")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]string{"status": "published"})
}

// --- Nested resource handlers ---

func (h *StoryHandler) UpsertRoom(w http.ResponseWriter, r *http.Request) {
	storyID, err := uuid.Parse(chi.URLParam(r, "storyId"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid story ID")
		return
	}
	roomID := chi.URLParam(r, "roomId")

	var req models.UpsertRoomRequest
	if err := DecodeJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.storyService.UpsertRoom(r.Context(), storyID, roomID, req); err != nil {
		h.handleError(w, r, err, "Failed to upsert room")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *StoryHandler) DeleteRoom(w http.ResponseWriter, r *http.Request) {
	storyID, err := uuid.Parse(chi.URLParam(r, "storyId"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid story ID")
		return
	}
	roomID := chi.URLParam(r, "roomId")

	if err := h.storyService.DeleteRoom(r.Context(), storyID, roomID); err != nil {
		h.handleError(w, r, err, "Failed to delete room")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *StoryHandler) UpsertItem(w http.ResponseWriter, r *http.Request) {
	storyID, err := uuid.Parse(chi.URLParam(r, "storyId"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid story ID")
		return
	}
	itemID := chi.URLParam(r, "itemId")

	var req models.UpsertItemRequest
	if err := DecodeJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.storyService.UpsertItem(r.Context(), storyID, itemID, req); err != nil {
		h.handleError(w, r, err, "Failed to upsert item")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *StoryHandler) DeleteItem(w http.ResponseWriter, r *http.Request) {
	storyID, err := uuid.Parse(chi.URLParam(r, "storyId"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid story ID")
		return
	}
	itemID := chi.URLParam(r, "itemId")

	if err := h.storyService.DeleteItem(r.Context(), storyID, itemID); err != nil {
		h.handleError(w, r, err, "Failed to delete item")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *StoryHandler) UpsertPuzzle(w http.ResponseWriter, r *http.Request) {
	storyID, err := uuid.Parse(chi.URLParam(r, "storyId"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid story ID")
		return
	}
	puzzleID := chi.URLParam(r, "puzzleId")

	var req models.UpsertPuzzleRequest
	if err := DecodeJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.storyService.UpsertPuzzle(r.Context(), storyID, puzzleID, req); err != nil {
		h.handleError(w, r, err, "Failed to upsert puzzle")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *StoryHandler) DeletePuzzle(w http.ResponseWriter, r *http.Request) {
	storyID, err := uuid.Parse(chi.URLParam(r, "storyId"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid story ID")
		return
	}
	puzzleID := chi.URLParam(r, "puzzleId")

	if err := h.storyService.DeletePuzzle(r.Context(), storyID, puzzleID); err != nil {
		h.handleError(w, r, err, "Failed to delete puzzle")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
