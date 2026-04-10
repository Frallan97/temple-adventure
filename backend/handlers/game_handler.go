package handlers

import (
	"log"
	"net/http"
	"strconv"

	"temple-adventure/models"
	"temple-adventure/services"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

type GameHandler struct {
	gameService *services.GameService
	debug       bool
}

func NewGameHandler(gameService *services.GameService, debug bool) *GameHandler {
	return &GameHandler{gameService: gameService, debug: debug}
}

func (h *GameHandler) handleError(w http.ResponseWriter, r *http.Request, err error, fallbackMsg string) {
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

func (h *GameHandler) CreateGame(w http.ResponseWriter, r *http.Request) {
	var req models.CreateGameRequest
	if err := DecodeJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.StoryID == uuid.Nil {
		WriteError(w, http.StatusBadRequest, "story_id is required")
		return
	}

	resp, err := h.gameService.CreateGame(r.Context(), req.StoryID)
	if err != nil {
		h.handleError(w, r, err, "Failed to create game")
		return
	}
	log.Printf("[GAME] Created session %s for story %s", resp.ID, req.StoryID)
	WriteJSON(w, http.StatusCreated, resp)
}

func (h *GameHandler) SendCommand(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	sessionID, err := uuid.Parse(idStr)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid game ID")
		return
	}

	var req models.CommandRequest
	if err := DecodeJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Input == "" {
		WriteError(w, http.StatusBadRequest, "Input is required")
		return
	}

	resp, err := h.gameService.ProcessCommand(r.Context(), sessionID, req.Input)
	if err != nil {
		log.Printf("[ERROR] [session:%s] [input:%q] ProcessCommand failed", sessionID, req.Input)
		h.handleError(w, r, err, "Failed to process command")
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *GameHandler) GetGame(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	sessionID, err := uuid.Parse(idStr)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid game ID")
		return
	}

	resp, err := h.gameService.GetGameState(r.Context(), sessionID)
	if err != nil {
		log.Printf("[ERROR] [session:%s] GetGameState failed", sessionID)
		h.handleError(w, r, err, "Failed to get game state")
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *GameHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	sessionID, err := uuid.Parse(idStr)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid game ID")
		return
	}

	limit := 50
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	resp, err := h.gameService.GetHistory(r.Context(), sessionID, limit, offset)
	if err != nil {
		log.Printf("[ERROR] [session:%s] GetHistory failed", sessionID)
		h.handleError(w, r, err, "Failed to get history")
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}
