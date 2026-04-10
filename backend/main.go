package main

import (
	"context"
	"log"
	"net/http"

	"temple-adventure/config"
	"temple-adventure/database"
	"temple-adventure/engine"
	"temple-adventure/handlers"
	"temple-adventure/services"
)

func main() {
	cfg := config.Load()

	if err := database.Connect(cfg.DatabaseURL); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	if err := database.RunMigrations(cfg.DatabaseURL); err != nil {
		log.Printf("Warning: Migration error: %v", err)
	}

	eng, err := engine.NewEngine(cfg.ContentDir)
	if err != nil {
		log.Fatalf("Failed to load game content: %v", err)
	}
	log.Printf("Game world loaded: %d rooms, %d items, %d puzzles",
		len(eng.World.Rooms), len(eng.World.Items), len(eng.World.Puzzles))

	// Seed default story into DB
	if err := services.SeedDefaultStory(context.Background(), database.DB, cfg.ContentDir); err != nil {
		log.Printf("Warning: Failed to seed default story: %v", err)
	}

	debug := cfg.Environment == "development"

	// Game service (with fallback engine for legacy sessions)
	gameService := services.NewGameService(database.DB, eng)
	gameHandler := handlers.NewGameHandler(gameService, debug)

	// Story service (shares engine cache with game service)
	storyRepo := services.NewStoryRepository(database.DB)
	storyService := services.NewStoryService(storyRepo, gameService.Cache())
	storyHandler := handlers.NewStoryHandler(storyService, debug)

	router := handlers.SetupRouter(cfg, gameHandler, storyHandler)

	addr := ":" + cfg.Port
	log.Printf("Starting server on %s (Environment: %s)", addr, cfg.Environment)

	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
