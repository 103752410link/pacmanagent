package game

// Point represents a position on the map
type Point struct {
	X, Y int
}

// FindPath uses BFS to find the shortest path from start to end on the map.
// Returns the path as a slice of Points (excluding start, including end).
// Returns empty slice if no path exists.
func FindPath(maze [][]int, start, end Point) []Point {
	if !isValid(maze, start.X, start.Y) || !isValid(maze, end.X, end.Y) {
		return nil
	}

	height := len(maze)
	width := len(maze[0])

	// steps[y][x] stores the previous cell that led to (x,y)
	steps := make([][]*Point, height)
	for i := range steps {
		steps[i] = make([]*Point, width)
	}

	dirs := []Point{{0, 1}, {1, 0}, {0, -1}, {-1, 0}}

	var find func([]Point) bool
	find = func(list []Point) bool {
		var newList []Point
		for _, current := range list {
			for _, d := range dirs {
				nx, ny := current.X+d.X, current.Y+d.Y
				if !isValid(maze, nx, ny) {
					// Tunnel wrap
					nx = (nx + width) % width
					ny = (ny + height) % height
				}
				if steps[ny][nx] != nil {
					continue
				}
				steps[ny][nx] = &current
				if nx == end.X && ny == end.Y {
					return true
				}
				newList = append(newList, Point{nx, ny})
			}
		}
		if len(newList) > 0 {
			return find(newList)
		}
		return false
	}

	if !find([]Point{start}) {
		return nil
	}

	// Reconstruct path
	var path []Point
	current := end
	for current.X != start.X || current.Y != start.Y {
		path = append([]Point{current}, path...)
		prev := steps[current.Y][current.X]
		if prev == nil {
			break
		}
		current = *prev
	}
	return path
}

// FindNextMoves returns valid adjacent moves from a position
func FindNextMoves(maze [][]int, pos Point) []Point {
	height := len(maze)
	width := len(maze[0])
	dirs := []Point{{0, 1}, {1, 0}, {0, -1}, {-1, 0}}
	var moves []Point
	for _, d := range dirs {
		nx, ny := pos.X+d.X, pos.Y+d.Y
		if !isValid(maze, nx, ny) {
			nx = (nx + width) % width
			ny = (ny + height) % height
		}
		if isValid(maze, nx, ny) || (nx >= 0 && nx < width && ny >= 0 && ny < height) {
			if maze[ny][nx] < 1 {
				moves = append(moves, Point{nx, ny})
			}
		}
	}
	return moves
}

func isValid(maze [][]int, x, y int) bool {
	if y < 0 || y >= len(maze) || x < 0 {
		return false
	}
	if x >= len(maze[y]) {
		return false // tunnel
	}
	return maze[y][x] < 1
}
