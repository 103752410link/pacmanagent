package game

// Player represents Pac-Man (controlled by an Agent)
type Player struct {
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	CoordX      int     `json:"coord_x"`
	CoordY      int     `json:"coord_y"`
	Orientation int     `json:"orientation"`
	Alive       bool    `json:"alive"`
}

// NewPlayer creates a new player at the default starting position
func NewPlayer(startX, startY int) *Player {
	return &Player{
		X:           float64(startX),
		Y:           float64(startY),
		CoordX:      startX,
		CoordY:      startY,
		Orientation: DirLeft,
		Alive:       true,
	}
}

// MoveOneStep moves the player one cell in the given direction
// Returns true if successfully moved, false if blocked
func (p *Player) MoveOneStep(direction int, maze [][]int) bool {
	if !p.Alive {
		return false
	}

	width := 28 // MapWidth
	nx := p.CoordX + Cos[direction]
	ny := p.CoordY + Sin[direction]

	// Handle tunnel
	if nx < 0 || nx >= width {
		nx = (nx + width) % width
	}

	if !p.isPassable(maze, nx, ny) {
		return false
	}

	p.Orientation = direction
	p.X = float64(p.CoordX + Cos[direction])
	p.Y = float64(p.CoordY + Sin[direction])

	// Tunnel wrap
	if p.X < 0 {
		p.X += 28
	} else if p.X >= 28 {
		p.X -= 28
	}

	p.CoordX = int(p.X)
	p.CoordY = int(p.Y)
	return true
}

func (p *Player) isPassable(maze [][]int, x, y int) bool {
	height := len(maze)
	width := 28 // MapWidth
	if y < 0 || y >= height {
		return false
	}
	if x < 0 || x >= width {
		return true // tunnel
	}
	return maze[y][x] < 1
}
