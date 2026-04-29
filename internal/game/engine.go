package game

import (
	"fmt"
	"sync"
	"time"
)

// TurnPhase represents the current phase of a turn
type TurnPhase int

const (
	PhaseWaiting TurnPhase = iota // Waiting for moves
)

// Bean represents a collectible bean on the map
type Bean struct {
	X, Y int
}

// GameState represents the current state of the game
type GameState struct {
	Round                int    `json:"round"`
	Phase                string `json:"phase"` // "player_turn" | "ghost_turn" | "game_over" | "restarting"
	PlayerMovesNeeded    int    `json:"player_moves_needed"`
	PlayerMovesSubmitted int    `json:"player_moves_submitted"`
	GhostMovesNeeded     int    `json:"ghost_moves_needed"`
	GhostMovesSubmitted  int    `json:"ghost_moves_submitted"`
	TimeoutRemaining     int    `json:"timeout_seconds_remaining"`
	Level                int    `json:"level"`
	Score                int    `json:"score"`
	Player               *Player `json:"player"`
	Ghosts               []*Ghost `json:"ghosts"`
	Beans                []Bean   `json:"beans"`
	BeansRemaining       int      `json:"beans_remaining"`
	GameOver             bool     `json:"game_over"`
	Running              bool     `json:"running"`
	Winner               string   `json:"winner,omitempty"`
	RestartCountdown     int      `json:"restart_countdown,omitempty"` // seconds until auto-restart
}

// Engine is the main game engine (turn-based)
type Engine struct {
	mu       sync.RWMutex
	maze     [][]int
	player   *Player
	ghosts   []*Ghost
	beans    map[string]bool
	score    int
	level    int
	round    int
	phase    TurnPhase // 0=waiting (player or ghost)
	running  bool

	// Turn state
	playerMoves    []string               // player's queued moves for this turn (max 2)
	ghostMoves     map[string][]string    // ghostID -> list of directions for this turn (max 2 each)
	ghostIDs       []string               // ordered list of ghost IDs
	ghostMovesPerRound int                 // how many moves each ghost submits per round

	// Timeout
	timeoutAt      time.Time
	timeoutSeconds int

	// Whether a player agent is registered (controls auto-advance)
	playerAgentRegistered bool

	// Auto restart
	restartCountdown int // seconds remaining until auto-restart
	restartTimer     *time.Timer

	// Callback
	OnStateChange func()
}

// NewEngine creates a new game engine
func NewEngine() *Engine {
	e := &Engine{
		maze:              deepCopyMap(MapData),
		score:             0,
		level:             0,
		round:             0,
		phase:             0,
		running:           false,
		ghostMoves:        make(map[string][]string),
		ghostMovesPerRound: 2,
		timeoutSeconds:    3,
	}
	e.initLevel()
	return e
}

func (e *Engine) initLevel() {
	e.player = NewPlayer(13, 23)
	e.beans = make(map[string]bool)

	for y := 0; y < MapHeight; y++ {
		for x := 0; x < MapWidth; x++ {
			if e.maze[y][x] == 0 {
				e.beans[beanKey(x, y)] = true
			}
		}
	}

	if e.level == 0 {
		e.beans["1,3"] = true
		e.beans["26,3"] = true
		e.beans["1,23"] = true
		e.beans["26,23"] = true
	}
}

// Start the game
func (e *Engine) Start() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.running = true
	e.round = 1
	e.restartCountdown = 0
	e.startNewTurn()
}

// Reset resets the game state for a new game (called during auto-restart)
func (e *Engine) Reset() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.running = false
	e.round = 0
	e.score = 0
	e.level = 0
	e.phase = 0
	e.playerMoves = nil
	e.ghostMoves = make(map[string][]string)
	e.restartCountdown = 0

	// Reset player
	e.player = NewPlayer(13, 23)

	// Reset beans
	e.beans = make(map[string]bool)
	for y := 0; y < MapHeight; y++ {
		for x := 0; x < MapWidth; x++ {
			if e.maze[y][x] == 0 {
				e.beans[beanKey(x, y)] = true
			}
		}
	}
	e.beans["1,3"] = true
	e.beans["26,3"] = true
	e.beans["1,23"] = true
	e.beans["26,23"] = true

	// Reset ghosts to starting positions
	for _, g := range e.ghosts {
		colorIdx := 0
		for i, gid := range e.ghostIDs {
			if gid == g.ID {
				colorIdx = i
				break
			}
		}
		startX := 11 + colorIdx%4
		startY := 11
		g.X = float64(startX)
		g.Y = float64(startY)
		g.CoordX = startX
		g.CoordY = startY
		g.Status = StatusNormal
		g.Timeout = 0
	}

	e.triggerStateChange()
}

// scheduleRestartLocked schedules an auto-restart after 60 seconds.
// Must be called with e.mu held.
func (e *Engine) scheduleRestartLocked() {
	const restartDelay = 60 // seconds
	e.restartCountdown = restartDelay

	if e.restartTimer != nil {
		e.restartTimer.Stop()
	}

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for i := restartDelay; i > 0; i-- {
			e.mu.Lock()
			e.restartCountdown = i
			e.triggerStateChange()
			e.mu.Unlock()
			<-ticker.C
		}

		e.Reset()
		e.Start()
		fmt.Println("[auto-restart] Game restarted!")
	}()
}

// SetGhostIDs sets the list of ghost IDs (called after ghosts are registered)
func (e *Engine) SetGhostIDs(ids []string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.ghostIDs = ids
	e.ghostMoves = make(map[string][]string)
}

func (e *Engine) startNewTurn() {
	e.playerMoves = nil
	e.ghostMoves = make(map[string][]string)
	e.timeoutAt = time.Now().Add(time.Duration(e.timeoutSeconds) * time.Second)
	e.phase = 0 // player turn
	e.triggerStateChange()
}

func (e *Engine) triggerStateChange() {
	if e.OnStateChange != nil {
		go e.OnStateChange()
	}
}

// SubmitPlayerMove adds a move for the player
func (e *Engine) SubmitPlayerMove(direction string) (bool, string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return false, "game not running"
	}

	e.playerAgentRegistered = true

	if e.phase != 0 {
		return false, "not player's turn"
	}
	if len(e.playerMoves) >= 2 {
		return false, "player already submitted 2 moves"
	}

	dir := directionNameToInt(direction)
	if dir < 0 && direction != "stay" {
		return false, "invalid direction"
	}

	e.playerMoves = append(e.playerMoves, direction)
	e.timeoutAt = time.Now().Add(time.Duration(e.timeoutSeconds) * time.Second) // reset timeout

	// Check if all player moves submitted
	if len(e.playerMoves) >= 2 {
		// Execute player phase
		e.executePlayerPhaseLocked()
		// Check if game ended during player phase (collision)
		if !e.running {
			e.triggerStateChange()
			return true, "game over"
		}
		// Move to ghost phase
		e.phase = 1
		e.ghostMoves = make(map[string][]string)
		e.timeoutAt = time.Now().Add(time.Duration(e.timeoutSeconds) * time.Second)
		e.triggerStateChange()
		return true, "player phase complete, waiting for ghosts"
	}

	e.triggerStateChange()
	return true, "move accepted"
}

// SubmitGhostMove adds a move for a ghost (up to ghostMovesPerRound per round)
func (e *Engine) SubmitGhostMove(agentID, direction string) (bool, string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return false, "game not running"
	}
	if e.phase != 1 {
		return false, "not ghost's turn"
	}

	// Check if this ghost exists
	found := false
	for _, id := range e.ghostIDs {
		if id == agentID {
			found = true
			break
		}
	}
	if !found {
		return false, "ghost not found"
	}

	// Check if this ghost already submitted enough moves
	if len(e.ghostMoves[agentID]) >= e.ghostMovesPerRound {
		return false, "ghost already submitted all moves for this round"
	}

	dir := directionNameToInt(direction)
	if dir < 0 && direction != "stay" {
		return false, "invalid direction"
	}

	e.ghostMoves[agentID] = append(e.ghostMoves[agentID], direction)
	e.timeoutAt = time.Now().Add(time.Duration(e.timeoutSeconds) * time.Second)

	// Check if all ghosts have submitted enough moves
	allDone := true
	for _, id := range e.ghostIDs {
		if len(e.ghostMoves[id]) < e.ghostMovesPerRound {
			allDone = false
			break
		}
	}

	if allDone {
		// Execute ghost phase
		e.executeGhostPhaseLocked()
		e.checkCollisionsLocked()
		e.checkBeanCollectionLocked()
		e.checkLevelCompleteLocked()

		if !e.running {
			e.triggerStateChange()
			return true, "round complete"
		}

		// Next round
		e.round++
		e.startNewTurnLocked()
		e.triggerStateChange()
		return true, "round complete, next round started"
	}

	e.triggerStateChange()
	return true, "move accepted"
}

// aiChaseDirection uses BFS to find the best direction for a ghost to chase the player.
// Returns a direction string ("right"|"down"|"left"|"up"|"stay").
func (e *Engine) aiChaseDirection(ghostID string) string {
	ghost := e.getGhostLocked(ghostID)
	if ghost == nil {
		return "stay"
	}

	// BFS from ghost position toward player
	startX, startY := ghost.CoordX, ghost.CoordY
	targetX, targetY := e.player.CoordX, e.player.CoordY

	// If scared, flee away from player
	if ghost.Status == StatusScared {
		targetX, targetY = 27-targetX, 14 // flee to opposite side
	}

	type step struct {
		x, y, dist int
		dir        int // first direction taken
	}

	dirs := []struct{ dx, dy, dir int }{
		{1, 0, DirRight},
		{0, 1, DirDown},
		{-1, 0, DirLeft},
		{0, -1, DirUp},
	}

	width := len(e.maze[0])
	height := len(e.maze)
	visited := make(map[string]bool)
	visited[fmt.Sprintf("%d,%d", startX, startY)] = true

	queue := []step{{startX, startY, 0, -1}}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		if cur.x == targetX && cur.y == targetY {
			return dirIntToName(cur.dir)
		}

		for _, d := range dirs {
			nx, ny := cur.x+d.dx, cur.y+d.dy
			// Tunnel wrap
			if nx < 0 || nx >= width {
				nx = (nx + width) % width
			}
			if ny < 0 || ny >= height {
				continue
			}
			k := fmt.Sprintf("%d,%d", nx, ny)
			if visited[k] {
				continue
			}
			if e.maze[ny][nx] >= 1 {
				continue
			}
			visited[k] = true
			firstDir := cur.dir
			if cur.dist == 0 {
				firstDir = d.dir
			}
			queue = append(queue, step{nx, ny, cur.dist + 1, firstDir})
		}
	}

	return "stay"
}

func dirIntToName(d int) string {
	switch d {
	case DirRight:
		return "right"
	case DirDown:
		return "down"
	case DirLeft:
		return "left"
	case DirUp:
		return "up"
	default:
		return "stay"
	}
}

// CheckTimeout checks if the current turn has timed out and auto-advances
func (e *Engine) CheckTimeout() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return
	}

	// No player agent registered — auto-advance immediately (AI mode)
	if !e.playerAgentRegistered {
		if e.phase == 0 {
			// Fill player moves with AI behavior
			for len(e.playerMoves) < 2 {
				e.playerMoves = append(e.playerMoves, "stay")
			}
			e.executePlayerPhaseLocked()
			e.phase = 1
			e.ghostMoves = make(map[string][]string)
			e.timeoutAt = time.Now().Add(time.Duration(e.timeoutSeconds) * time.Second)
			e.triggerStateChange()
			return
		}
		// Ghost phase — auto-advance with AI chase
		for _, id := range e.ghostIDs {
			for len(e.ghostMoves[id]) < e.ghostMovesPerRound {
				e.ghostMoves[id] = append(e.ghostMoves[id], e.aiChaseDirection(id))
			}
		}
		e.executeGhostPhaseLocked()
		e.checkCollisionsLocked()
		e.checkBeanCollectionLocked()
		e.checkLevelCompleteLocked()
		if !e.running {
			e.triggerStateChange()
			return
		}
		e.round++
		e.startNewTurnLocked()
		e.triggerStateChange()
		return
	}

	if time.Now().Before(e.timeoutAt) {
		return
	}

	// Timeout - auto-advance
	if e.phase == 0 {
		// Player timeout - fill remaining moves with "stay"
		for len(e.playerMoves) < 2 {
			e.playerMoves = append(e.playerMoves, "stay")
		}
		e.executePlayerPhaseLocked()
		e.phase = 1
		e.ghostMoves = make(map[string][]string)
		e.timeoutAt = time.Now().Add(time.Duration(e.timeoutSeconds) * time.Second)
		e.triggerStateChange()
	} else if e.phase == 1 {
		// Ghost timeout - fill missing ghost moves with AI chase
		for _, id := range e.ghostIDs {
			for len(e.ghostMoves[id]) < e.ghostMovesPerRound {
				e.ghostMoves[id] = append(e.ghostMoves[id], e.aiChaseDirection(id))
			}
		}
		e.executeGhostPhaseLocked()
		e.checkCollisionsLocked()
		e.checkBeanCollectionLocked()
		e.checkLevelCompleteLocked()

		if !e.running {
			e.triggerStateChange()
			return
		}

		e.round++
		e.startNewTurnLocked()
		e.triggerStateChange()
	}
}

func (e *Engine) startNewTurnLocked() {
	e.playerMoves = nil
	e.ghostMoves = make(map[string][]string)
	e.timeoutAt = time.Now().Add(time.Duration(e.timeoutSeconds) * time.Second)
	e.phase = 0
}

func (e *Engine) executePlayerPhaseLocked() {
	// Execute each player move
	for _, dir := range e.playerMoves {
		d := directionNameToInt(dir)
		if d >= 0 {
			e.player.MoveOneStep(d, e.maze)
		}
		// Check bean collection after each move
		e.checkBeanCollectionLocked()
		// Check collision after each move (player could walk into a ghost)
		e.checkCollisionsLocked()
		if !e.running {
			return
		}
	}
}

func (e *Engine) executeGhostPhaseLocked() {
	for _, id := range e.ghostIDs {
		moves, ok := e.ghostMoves[id]
		if !ok || len(moves) == 0 {
			continue // ghost didn't submit, stay in place
		}

		ghost := e.getGhostLocked(id)
		if ghost == nil {
			continue
		}

		for _, dir := range moves {
			d := directionNameToInt(dir)
			if d >= 0 {
				ghost.MoveOneStep(d, e.maze)
			}
			// Check collision and bean collection after each ghost step
			e.checkCollisionsLocked()
			e.checkBeanCollectionLocked()
			if !e.running {
				return
			}
		}
	}
}

func (e *Engine) checkBeanCollectionLocked() {
	key := beanKey(e.player.CoordX, e.player.CoordY)
	if e.beans[key] {
		delete(e.beans, key)
		e.score++

		// Power pellet
		if e.level == 0 && (key == "1,3" || key == "26,3" || key == "1,23" || key == "26,23") {
			for _, g := range e.ghosts {
				if g.Status == StatusNormal {
					g.Status = StatusScared
					g.Timeout = 450
				}
			}
		}
	}
}

func (e *Engine) checkCollisionsLocked() {
	for _, g := range e.ghosts {
		// Exact cell collision (grid-based)
		if g.CoordX == e.player.CoordX && g.CoordY == e.player.CoordY {
			if g.Status == StatusScared {
				fmt.Printf("[collision] Player ate %s ghost at (%d,%d)\n", g.Name, g.CoordX, g.CoordY)
				g.Status = StatusReturning
				g.Timeout = 0
				e.score += 10
			} else if g.Status == StatusNormal {
				fmt.Printf("[collision] %s caught player at (%d,%d) - GAME OVER\n", g.Name, g.CoordX, g.CoordY)
				e.player.Alive = false
				e.running = false
				e.scheduleRestartLocked()
				e.triggerStateChange()
				return
			}
		}
	}
}

func (e *Engine) checkLevelCompleteLocked() {
	if len(e.beans) == 0 {
		e.level++
		if e.level >= 12 {
			e.running = false
			e.scheduleRestartLocked()
		} else {
			fmt.Printf("[level] Level %d complete, advancing to level %d\n", e.level, e.level+1)
			e.initLevel()
			// Reset all ghosts to starting positions with Normal status
			for i, g := range e.ghosts {
				startX := 11 + i%4
				startY := 11
				g.X = float64(startX)
				g.Y = float64(startY)
				g.CoordX = startX
				g.CoordY = startY
				g.Status = StatusNormal
				g.Timeout = 0
				g.Orientation = DirUp
			}
		}
	}
}

func (e *Engine) getGhostLocked(id string) *Ghost {
	for _, g := range e.ghosts {
		if g.ID == id {
			return g
		}
	}
	return nil
}

// RegisterGhost adds a new ghost to the game
func (e *Engine) RegisterGhost(id, name string) (*Ghost, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	colorIdx := len(e.ghosts) % len(GhostColors)
	color := GhostColors[colorIdx]

	startX := 11 + colorIdx%4
	startY := 11

	ghost := NewGhost(id, name, color, startX, startY)
	e.ghosts = append(e.ghosts, ghost)
	e.ghostIDs = append(e.ghostIDs, id)
	return ghost, nil
}

// SetPlayerAgentRegistered marks that a player agent has registered
func (e *Engine) SetPlayerAgentRegistered() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.playerAgentRegistered = true
	// Reset timeout so player has full 30 seconds from registration
	e.timeoutAt = time.Now().Add(time.Duration(e.timeoutSeconds) * time.Second)
}

// UnregisterGhost removes a ghost
func (e *Engine) UnregisterGhost(id string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	for i, g := range e.ghosts {
		if g.ID == id {
			e.ghosts = append(e.ghosts[:i], e.ghosts[i+1:]...)
			// Remove from ghostIDs
			for j, gid := range e.ghostIDs {
				if gid == id {
					e.ghostIDs = append(e.ghostIDs[:j], e.ghostIDs[j+1:]...)
					break
				}
			}
			return true
		}
	}
	return false
}

// GetGhost returns a ghost by ID
func (e *Engine) GetGhost(id string) *Ghost {
	e.mu.RLock()
	defer e.mu.RUnlock()
	for _, g := range e.ghosts {
		if g.ID == id {
			return g
		}
	}
	return nil
}

// GetState returns the current game state
func (e *Engine) GetState() GameState {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var beans []Bean
	for key := range e.beans {
		var x, y int
		_, err := fmt.Sscanf(key, "%d,%d", &x, &y)
		if err == nil {
			beans = append(beans, Bean{X: x, Y: y})
		}
	}

	phase := "player_turn"
	if e.phase == 1 {
		phase = "ghost_turn"
	} else if !e.playerAgentRegistered {
		phase = "waiting_for_player"
	}

	timeoutRemaining := 0
	if !e.timeoutAt.IsZero() {
		remaining := int(e.timeoutAt.Sub(time.Now()).Seconds())
		if remaining > 0 {
			timeoutRemaining = remaining
		}
	}

	// Count total ghost moves submitted across all ghosts
	ghostMovesSubmitted := 0
	for _, moves := range e.ghostMoves {
		ghostMovesSubmitted += len(moves)
	}

	// Determine phase string
	if !e.running {
		if e.restartCountdown > 0 {
			phase = "restarting"
		} else {
			phase = "game_over"
		}
	}

	return GameState{
		Round:                e.round,
		Phase:                phase,
		PlayerMovesNeeded:    2,
		PlayerMovesSubmitted: len(e.playerMoves),
		GhostMovesNeeded:     len(e.ghostIDs) * e.ghostMovesPerRound,
		GhostMovesSubmitted:  ghostMovesSubmitted,
		TimeoutRemaining:     timeoutRemaining,
		Level:                e.level + 1,
		Score:                e.score,
		Player:               e.player,
		Ghosts:               e.ghosts,
		Beans:                beans,
		BeansRemaining:       len(e.beans),
		GameOver:             !e.player.Alive || e.level >= 12,
		Running:              e.running,
		RestartCountdown:     e.restartCountdown,
	}
}

// GetMaze returns the current maze
func (e *Engine) GetMaze() [][]int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.maze
}

// GetWallColor returns the wall color for the current level
func (e *Engine) GetWallColor() string {
	return WallColors[e.level%len(WallColors)]
}

func beanKey(x, y int) string {
	return fmt.Sprintf("%d,%d", x, y)
}

func deepCopyMap(src [][]int) [][]int {
	dst := make([][]int, len(src))
	for i := range src {
		dst[i] = make([]int, len(src[i]))
		copy(dst[i], src[i])
	}
	return dst
}

func directionNameToInt(dir string) int {
	switch dir {
	case "right":
		return DirRight
	case "down":
		return DirDown
	case "left":
		return DirLeft
	case "up":
		return DirUp
	case "stay":
		return -1 // will result in no movement
	default:
		return -1
	}
}
