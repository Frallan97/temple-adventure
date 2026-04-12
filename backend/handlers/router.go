package handlers

import (
	"net/http"

	"temple-adventure/config"
	"temple-adventure/middleware"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
)

func SetupRouter(cfg *config.Config, gameHandler *GameHandler, storyHandler *StoryHandler, specHandler *SpecHandler) http.Handler {
	r := chi.NewRouter()

	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Recoverer)
	r.Use(middleware.LoggingMiddleware)
	r.Use(middleware.CORSMiddleware(cfg.AllowedOrigins))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Route("/api/v1", func(r chi.Router) {
		// Game endpoints
		r.Get("/games/logs", gameHandler.GetGameLogs)
		r.Post("/games", gameHandler.CreateGame)
		r.Get("/games/{id}", gameHandler.GetGame)
		r.Post("/games/{id}/command", gameHandler.SendCommand)
		r.Get("/games/{id}/history", gameHandler.GetHistory)

		// Story endpoints
		r.Get("/stories", storyHandler.ListStories)
		r.Post("/stories", storyHandler.CreateStory)
		r.Post("/stories/from-spec", specHandler.CreateFromSpec)
		r.Route("/stories/{storyId}", func(r chi.Router) {
			r.Get("/", storyHandler.GetStory)
			r.Put("/", storyHandler.UpdateStory)
			r.Delete("/", storyHandler.DeleteStory)
			r.Post("/validate", storyHandler.ValidateStory)
			r.Post("/publish", storyHandler.PublishStory)
			r.Post("/ratings", storyHandler.RateStory)
			r.Get("/ratings", storyHandler.GetStoryRating)

			r.Put("/rooms/{roomId}", storyHandler.UpsertRoom)
			r.Delete("/rooms/{roomId}", storyHandler.DeleteRoom)
			r.Put("/items/{itemId}", storyHandler.UpsertItem)
			r.Delete("/items/{itemId}", storyHandler.DeleteItem)
			r.Put("/puzzles/{puzzleId}", storyHandler.UpsertPuzzle)
			r.Delete("/puzzles/{puzzleId}", storyHandler.DeletePuzzle)
			r.Put("/npcs/{npcId}", storyHandler.UpsertNpc)
			r.Delete("/npcs/{npcId}", storyHandler.DeleteNpc)
		})
	})

	return r
}
