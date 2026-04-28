package server

import (
	"fmt"
	"net/http"

	"pacman/internal/agent"
	"pacman/internal/game"

	"github.com/gin-gonic/gin"
)

// Handler holds references to the game engine and agent registry
type Handler struct {
	Engine   *game.Engine
	Registry *agent.Registry
	hub      *Hub
}

// NewHandler creates a new Handler
func NewHandler(engine *game.Engine, registry *agent.Registry) *Handler {
	hub := NewHub()
	go hub.Run()
	return &Handler{
		Engine:   engine,
		Registry: registry,
		hub:      hub,
	}
}

// RegisterRoutes sets up all HTTP routes
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	// Agent management
	r.POST("/api/agent/register", h.registerAgent)
	r.DELETE("/api/agent/:id", h.unregisterAgent)
	r.POST("/api/agent/:id/action", h.agentAction)

	// Game control
	r.POST("/api/game/start", h.startGame)
	r.POST("/api/game/pause", h.pauseGame)
	r.GET("/api/game/state", h.getGameState)
	r.GET("/api/game/maze", h.getMaze)

	// WebSocket
	r.GET("/ws", h.websocketHandler)

	// Serve static files
	r.Static("/static", "./static")
	r.LoadHTMLFiles("index.html")
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})
}

// Broadcast sends a message to all WebSocket clients
func (h *Handler) Broadcast(message []byte) {
	h.hub.Broadcast(message)
}

// MoveRequest for agent move submission
type MoveRequest struct {
	Direction string `json:"direction" binding:"required,oneof=left right up down stay"`
}

// MoveResponse with turn-based state
type MoveResponse struct {
	Success              bool   `json:"success"`
	Message              string `json:"message"`
	Round                int    `json:"round"`
	Phase                string `json:"phase"`
	PlayerMovesSubmitted int    `json:"player_moves_submitted"`
	GhostMovesSubmitted  int    `json:"ghost_moves_submitted"`
}

// RegisterRequest for agent registration
type RegisterRequest struct {
	Name     string `json:"name" binding:"required"`
	Strategy string `json:"strategy"`
}

func (h *Handler) registerAgent(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	h.doRegister(c, req.Name, req.Strategy)
}

func (h *Handler) doRegister(c *gin.Context, name, strategy string) {
	a := h.Registry.Register(name, strategy)

	var color string
	var role string
	if a.Role == agent.RolePlayer {
		color = "yellow"
		role = "player"
		h.Engine.SetPlayerAgentRegistered()
	} else {
		ghost, err := h.Engine.RegisterGhost(a.ID, a.Name)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to register ghost"})
			return
		}
		color = ghost.Color
		role = "ghost"

		// Sync ghost IDs with engine
		ghostAgents := h.Registry.GetGhosts()
		var ghostIDs []string
		for _, ga := range ghostAgents {
			ghostIDs = append(ghostIDs, ga.ID)
		}
		h.Engine.SetGhostIDs(ghostIDs)
	}

	c.JSON(http.StatusOK, gin.H{
		"agent_id": a.ID,
		"role":     role,
		"color":    color,
		"message":  "Agent registered as " + role,
	})
}

func (h *Handler) unregisterAgent(c *gin.Context) {
	id := c.Param("id")
	a := h.Registry.Get(id)
	if a != nil && a.Role == agent.RoleGhost {
		h.Engine.UnregisterGhost(id)
		// Re-sync ghost IDs
		ghostAgents := h.Registry.GetGhosts()
		var ghostIDs []string
		for _, ga := range ghostAgents {
			ghostIDs = append(ghostIDs, ga.ID)
		}
		h.Engine.SetGhostIDs(ghostIDs)
	}
	h.Registry.Unregister(id)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *Handler) agentAction(c *gin.Context) {
	id := c.Param("id")

	a := h.Registry.Get(id)
	if a == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	var req MoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Printf("[handler] agentAction(%s) bind error: %v\n", id, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var success bool
	var msg string

	if a.Role == agent.RolePlayer {
		success, msg = h.Engine.SubmitPlayerMove(req.Direction)
	} else {
		success, msg = h.Engine.SubmitGhostMove(id, req.Direction)
	}

	if !success {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	state := h.Engine.GetState()
	c.JSON(http.StatusOK, MoveResponse{
		Success:              true,
		Message:              msg,
		Round:                state.Round,
		Phase:                state.Phase,
		PlayerMovesSubmitted: state.PlayerMovesSubmitted,
		GhostMovesSubmitted:  state.GhostMovesSubmitted,
	})
}

func (h *Handler) startGame(c *gin.Context) {
	h.Engine.Start()
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Game started"})
}

func (h *Handler) pauseGame(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "paused"})
}

func (h *Handler) getGameState(c *gin.Context) {
	state := h.Engine.GetState()
	c.JSON(http.StatusOK, state)
}

func (h *Handler) getMaze(c *gin.Context) {
	maze := h.Engine.GetMaze()
	c.JSON(http.StatusOK, gin.H{
		"maze":       maze,
		"wall_color": h.Engine.GetWallColor(),
	})
}
