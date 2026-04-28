package game

// Direction constants (matching JS: 0=right, 1=down, 2=left, 3=up)
const (
	DirRight = 0
	DirDown  = 1
	DirLeft  = 2
	DirUp    = 3
)

// Direction deltas: [x, y]
var Cos = []int{1, 0, -1, 0}
var Sin = []int{0, 1, 0, -1}

// GhostStatus constants
const (
	StatusInactive  = 0
	StatusNormal    = 1
	StatusPaused    = 2
	StatusScared    = 3 // weakened by power pellet
	StatusReturning = 4 // returning to base after being eaten
)

// Ghost represents an AI-controlled ghost
type Ghost struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Color           string  `json:"color"`
	X               float64 `json:"x"`
	Y               float64 `json:"y"`
	CoordX          int     `json:"coord_x"`
	CoordY          int     `json:"coord_y"`
	Orientation     int     `json:"orientation"`
	Status          int     `json:"status"`
	Timeout         int     `json:"timeout"`    // countdown for scared state
	desiredDir      int     // agent's desired direction (-1 = no preference)
	moveProgress    float64 // progress toward next cell (0..1)
	returnPath      []Point // path when returning to base
}

// NewGhost creates a new ghost at the given position
func NewGhost(id, name, color string, startX, startY int) *Ghost {
	return &Ghost{
		ID:          id,
		Name:        name,
		Color:       color,
		X:           float64(startX),
		Y:           float64(startY),
		CoordX:      startX,
		CoordY:      startY,
		Orientation: DirUp,
		Status:      StatusNormal,
		Timeout:     0,
		desiredDir:  -1,
	}
}

// QueueCommand stores the agent's desired direction
func (g *Ghost) QueueCommand(direction string) {
	switch direction {
	case "right":
		g.desiredDir = DirRight
	case "down":
		g.desiredDir = DirDown
	case "left":
		g.desiredDir = DirLeft
	case "up":
		g.desiredDir = DirUp
	}
}

// MoveOneStep moves the ghost one cell in the given direction.
// If that direction is blocked, tries other directions.
func (g *Ghost) MoveOneStep(direction int, maze [][]int) bool {
	if g.Status == StatusScared {
		g.Timeout--
		if g.Timeout <= 0 {
			g.Status = StatusNormal
		}
	}
	if g.Status == StatusReturning {
		g.updateReturn(maze)
		return true
	}

	width := len(maze[0])

	// Try directions: desired, current, then others
	dirs := []int{}
	if direction >= 0 {
		dirs = append(dirs, direction)
	}
	if g.Orientation >= 0 {
		found := false
		for _, d := range dirs {
			if d == g.Orientation {
				found = true
				break
			}
		}
		if !found {
			dirs = append(dirs, g.Orientation)
		}
	}
	for _, d := range []int{DirRight, DirDown, DirLeft, DirUp} {
		found := false
		for _, existing := range dirs {
			if d == existing {
				found = true
				break
			}
		}
		if !found {
			dirs = append(dirs, d)
		}
	}

	for _, dir := range dirs {
		nx := g.CoordX + Cos[dir]
		ny := g.CoordY + Sin[dir]
		if nx < 0 || nx >= width {
			nx = (nx + width) % width
		}
		if g.canMove(maze, nx, ny) {
			g.Orientation = dir
			g.X = float64(g.CoordX + Cos[dir])
			g.Y = float64(g.CoordY + Sin[dir])
			if g.X < 0 {
				g.X += float64(width)
			} else if g.X >= float64(width) {
				g.X -= float64(width)
			}
			g.CoordX = int(g.X)
			g.CoordY = int(g.Y)
			if g.CoordX < 0 || g.CoordX >= width {
				g.CoordX = (g.CoordX + width) % width
			}
			return true
		}
	}
	return false
}

// Update moves the ghost one cell per tick
// 1. Try agent's desired direction
// 2. Fall back to current direction
// 3. If both blocked, try other directions
func (g *Ghost) Update(maze [][]int) {
	if g.Status == StatusScared {
		g.Timeout--
		if g.Timeout <= 0 {
			g.Status = StatusNormal
		}
	}

	if g.Status == StatusReturning {
		g.updateReturn(maze)
		return
	}

	width := len(maze[0])

	// Try directions in order of preference
	dirs := g.getDirections()
	moved := false

	for _, dir := range dirs {
		nx := g.CoordX + Cos[dir]
		ny := g.CoordY + Sin[dir]

		// Handle tunnel
		if nx < 0 || nx >= width {
			nx = (nx + width) % width
		}

		if g.canMove(maze, nx, ny) {
			g.Orientation = dir
			g.X = float64(g.CoordX + Cos[dir])
			g.Y = float64(g.CoordY + Sin[dir])

			// Tunnel wrap
			if g.X < 0 {
				g.X += float64(width)
			} else if g.X >= float64(width) {
				g.X -= float64(width)
			}

			g.CoordX = int(g.X)
			g.CoordY = int(g.Y)
			if g.CoordX < 0 || g.CoordX >= width {
				g.CoordX = (g.CoordX + width) % width
			}

			moved = true
			break
		}
	}

	if !moved && g.Status == StatusNormal {
		// Try to find any valid move
		for _, dir := range []int{DirRight, DirDown, DirLeft, DirUp} {
			nx := g.CoordX + Cos[dir]
			ny := g.CoordY + Sin[dir]
			if nx < 0 || nx >= width {
				nx = (nx + width) % width
			}
			if g.canMove(maze, nx, ny) {
				g.Orientation = dir
				g.X = float64(g.CoordX + Cos[dir])
				g.Y = float64(g.CoordY + Sin[dir])
				if g.X < 0 {
					g.X += float64(width)
				} else if g.X >= float64(width) {
					g.X -= float64(width)
				}
				g.CoordX = int(g.X)
				g.CoordY = int(g.Y)
				if g.CoordX < 0 || g.CoordX >= width {
					g.CoordX = (g.CoordX + width) % width
				}
				break
			}
		}
	}
}

// getDirections returns preferred direction order
func (g *Ghost) getDirections() []int {
	if g.Status == StatusScared {
		// Scared: try random directions
		return []int{(g.Orientation + 2) % 4, (g.Orientation + 1) % 4, (g.Orientation + 3) % 4, g.Orientation}
	}

	// Normal: prioritize desired direction, then current, then others
	if g.desiredDir >= 0 {
		others := []int{}
		for _, d := range []int{DirRight, DirDown, DirLeft, DirUp} {
			if d != g.desiredDir && d != g.Orientation {
				others = append(others, d)
			}
		}
		return []int{g.desiredDir, g.Orientation, others[0], others[1]}
	}

	// No desired direction: continue current, then try others
	return []int{g.Orientation, (g.Orientation + 1) % 4, (g.Orientation + 3) % 4, (g.Orientation + 2) % 4}
}

func (g *Ghost) canMove(maze [][]int, x, y int) bool {
	height := len(maze)
	if y < 0 || y >= height {
		return false
	}
	if x < 0 || x >= len(maze[y]) {
		return false // out of bounds
	}
	// Ghosts can pass through empty cells (value 0)
	return maze[y][x] < 1
}

// updateReturn moves ghost back to base using BFS
func (g *Ghost) updateReturn(maze [][]int) {
	targetX := 13
	targetY := 14

	if g.CoordX == targetX && g.CoordY == targetY {
		g.Status = StatusNormal
		g.X = float64(targetX)
		g.Y = float64(targetY)
		g.CoordX = targetX
		g.CoordY = targetY
		g.Orientation = DirUp
		return
	}

	// BFS to find shortest path to base
	type step struct {
		x, y int
		dir  int // direction taken to reach this cell
	}
	visited := make(map[string]bool)
	queue := []step{{g.CoordX, g.CoordY, -1}}
	visited[key(g.CoordX, g.CoordY)] = true

	parent := make(map[string]step)
	found := false

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		if cur.x == targetX && cur.y == targetY {
			found = true
			break
		}

		for _, dir := range []int{DirRight, DirDown, DirLeft, DirUp} {
			nx := cur.x + Cos[dir]
			ny := cur.y + Sin[dir]
			w := len(maze[0])
			if nx < 0 || nx >= w {
				nx = (nx + w) % w
			}
			k := key(nx, ny)
			if !visited[k] && g.canMove(maze, nx, ny) {
				visited[k] = true
				parent[k] = step{cur.x, cur.y, dir}
				queue = append(queue, step{nx, ny, dir})
			}
		}
	}

	if !found {
		// Fallback: try any valid move
		for _, dir := range []int{DirRight, DirDown, DirLeft, DirUp} {
			nx := g.CoordX + Cos[dir]
			ny := g.CoordY + Sin[dir]
			w := len(maze[0])
			if nx < 0 || nx >= w {
				nx = (nx + w) % w
			}
			if g.canMove(maze, nx, ny) {
				g.Orientation = dir
				g.X = float64(nx)
				g.Y = float64(ny)
				g.CoordX = nx
				g.CoordY = ny
				return
			}
		}
		return
	}

	// Backtrack from target to find first direction
	k := key(targetX, targetY)
	firstDir := parent[k].dir
	for parent[k].x != g.CoordX || parent[k].y != g.CoordY {
		pk := key(parent[k].x, parent[k].y)
		firstDir = parent[pk].dir
		k = pk
	}

	g.Orientation = firstDir
	nx := g.CoordX + Cos[firstDir]
	ny := g.CoordY + Sin[firstDir]
	w := len(maze[0])
	if nx < 0 || nx >= w {
		nx = (nx + w) % w
	}
	g.X = float64(nx)
	g.Y = float64(ny)
	g.CoordX = nx
	g.CoordY = ny
}

func key(x, y int) string {
	return string(rune(x*1000 + y))
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
