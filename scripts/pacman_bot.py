#!/usr/bin/env python3
"""Pac-Man Agent - Smart pathfinder: eat nearest bean, avoid ghosts."""
import requests, time, sys

BASE = "http://localhost:8080"
OPPOSITE = {"right": "left", "left": "right", "up": "down", "down": "up"}
DIRS = ["right", "down", "left", "up"]

def get_maze():
    return requests.get(f"{BASE}/api/game/maze").json()["maze"]

def get_state():
    return requests.get(f"{BASE}/api/game/state").json()

def submit_move(agent_id, direction):
    return requests.post(f"{BASE}/api/agent/{agent_id}/action", json={"direction": direction}).json()

def nearest_bean(px, py, beans, maze):
    """Find nearest bean using BFS."""
    if not beans:
        return None
    visited = set()
    queue = [(px, py, [])]
    visited.add((px, py))
    beans_set = set((b["X"], b["Y"]) for b in beans)
    while queue:
        x, y, path = queue.pop(0)
        if (x, y) in beans_set:
            return path
        for d, (dx, dy) in enumerate([(1,0),(0,1),(-1,0),(0,-1)]):
            nx, ny = x + dx, y + dy
            if 0 <= nx < 28 and 0 <= ny < len(maze) and maze[ny][nx] < 1 and (nx, ny) not in visited:
                visited.add((nx, ny))
                queue.append((nx, ny, path + [DIRS[d]]))
    return None

def is_ghost_near(px, py, ghosts, radius=3):
    """Check if any normal ghost is nearby."""
    for g in ghosts:
        if g["status"] in (1, 3, 4):
            dist = abs(g["coord_x"] - px) + abs(g["coord_y"] - py)
            if dist <= radius:
                return True
    return False

def run(agent_id):
    print(f"[Pac-Man] Agent {agent_id}")
    last_dir = "left"
    while True:
        try:
            state = get_state()
            if state.get("game_over"):
                print(f"[Pac-Man] Game over. Score: {state['score']}")
                break

            if state["phase"] != "player_turn":
                time.sleep(0.1)
                continue

            if state.get("player_moves_submitted", 0) >= 2:
                time.sleep(0.1)
                continue

            px = state["player"]["coord_x"]
            py = state["player"]["coord_y"]
            maze = get_maze()
            ghosts = state.get("ghosts") or []
            beans = state.get("beans") or []

            # Compute 2 moves
            moves = []
            for _ in range(2):
                if is_ghost_near(px, py, ghosts, radius=4):
                    # Flee: pick direction away from nearest ghost
                    best_dir = None
                    best_dist = -1
                    for d in DIRS:
                        if moves and d == OPPOSITE[moves[-1]]:
                            continue
                        dx = {"right":1,"left":-1,"up":0,"down":0}[d]
                        dy = {"right":0,"left":0,"up":-1,"down":1}[d]
                        nx, ny = px + dx, py + dy
                        if 0 <= nx < 28 and 0 <= ny < len(maze) and maze[ny][nx] < 1:
                            # Min distance to nearest ghost
                            min_dist = 999
                            for g in ghosts:
                                dist = abs(g["coord_x"] - nx) + abs(g["coord_y"] - ny)
                                if dist < min_dist:
                                    min_dist = dist
                            if min_dist > best_dist:
                                best_dist = min_dist
                                best_dir = d
                    moves.append(best_dir or last_dir)
                else:
                    path = nearest_bean(px, py, beans, maze)
                    if path:
                        first = path[0]
                        if moves and first == OPPOSITE[moves[-1]]:
                            path = path[1:] if len(path) > 1 else []
                            if path:
                                moves.append(path[0])
                            else:
                                moves.append(last_dir)
                        else:
                            moves.append(first)
                    else:
                        moves.append(last_dir)

                # Update position for second move
                if moves[-1]:
                    px += {"right":1,"left":-1,"up":0,"down":0}.get(moves[-1], 0)
                    py += {"right":0,"left":0,"up":-1,"down":1}.get(moves[-1], 0)
                if px < 0: px = 27
                if px > 27: px = 0

            last_dir = moves[-1] if moves else last_dir

            results = []
            for m in moves:
                r = submit_move(agent_id, m)
                results.append(f"{m}->{r.get('message','?')}")
            print(f"[Pac-Man] Round {state['round']}, Score {state['score']}: {', '.join(results)}")

        except Exception as e:
            print(f"[Pac-Man] Error: {e}")
        time.sleep(0.3)

if __name__ == "__main__":
    # Register first
    resp = requests.post(f"{BASE}/api/agent/register", json={"name": "PacBot"})
    data = resp.json()
    print(f"Registered: {data}")
    run(data["agent_id"])
