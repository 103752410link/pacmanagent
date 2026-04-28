package main

import (
	"encoding/json"
	"log"
	"time"

	"pacman/internal/agent"
	"pacman/internal/game"
	"pacman/internal/server"

	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize game engine and agent registry
	engine := game.NewEngine()
	registry := agent.NewRegistry()
	handler := server.NewHandler(engine, registry)

	// Broadcast game state on every state change
	engine.OnStateChange = func() {
		state := engine.GetState()
		data, err := json.Marshal(state)
		if err == nil {
			handler.Broadcast(data)
		}
	}

	// Timeout checker — runs every second
	go func() {
		for {
			time.Sleep(1 * time.Second)
			engine.CheckTimeout()
		}
	}()

	// Auto-start the game
	engine.Start()

	// Setup Gin router
	r := gin.Default()
	handler.RegisterRoutes(r)

	log.Println("Pac-Man Agent Arena starting on :8080")
	log.Println("Open browser: http://localhost:8080")
	log.Println("API endpoints:")
	log.Println("  POST /api/agent/register  - Register agent (first=player, rest=ghosts)")
	log.Println("  POST /api/agent/:id/action - Submit move (direction: left|right|up|down)")
	log.Println("  DELETE /api/agent/:id      - Unregister agent")
	log.Println("  GET  /api/game/state       - Get current game state")
	log.Println("  WS   /ws                   - WebSocket for real-time state")

	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
