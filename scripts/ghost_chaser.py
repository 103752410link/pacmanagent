#!/usr/bin/env python3
"""Ghost Agent - Chaser: pursues the player directly."""
import requests, time, sys

BASE = "http://localhost:8080"
DIRS = ["right", "down", "left", "up"]

def get_state():
    return requests.get(f"{BASE}/api/game/state").json()

def get_maze():
    return requests.get(f"{BASE}/api/game/maze").json()["maze"]

def submit_move(agent_id, direction):
    return requests.post(f"{BASE}/api/agent/{agent_id}/action", json={"direction": direction}).json()

def bfs_to_target(start_x, start_y, target_x, target_y, maze):
    """BFS to find first step toward target."""
    if start_x == target_x and start_y == target_y:
        return "stay"
    visited = set()
    queue = [(start_x, start_y, [])]
    visited.add((start_x, start_y))
    while queue:
        x, y, path = queue.pop(0)
        if x == target_x and y == target_y:
            return path[0] if path else "stay"
        for i, (dx, dy) in enumerate([(1,0),(0,1),(-1,0),(0,-1)]):
            nx, ny = x + dx, y + dy
            if 0 <= nx < 28 and 0 <= ny < len(maze) and maze[ny][nx] < 1 and (nx, ny) not in visited:
                visited.add((nx, ny))
                queue.append((nx, ny, path + [DIRS[i]]))
    return "stay"

def run(agent_id):
    my_id = agent_id
    print(f"[Chaser Ghost] Agent {agent_id}")
    while True:
        try:
            state = get_state()
            if state.get("game_over"):
                print(f"[Ghost] Game over. Score: {state['score']}")
                break

            if state["phase"] != "ghost_turn":
                time.sleep(0.1)
                continue

            maze = get_maze()
            player = state["player"]
            px, py = player["coord_x"], player["coord_y"]

            # Find my position in state ghosts
            my_ghost = None
            for g in state.get("ghosts") or []:
                if g["id"] == agent_id:
                    my_ghost = g
                    break
            if not my_ghost:
                time.sleep(0.2)
                continue

            gx, gy = my_ghost["coord_x"], my_ghost["coord_y"]
            direction = bfs_to_target(gx, gy, px, py, maze)
            r = submit_move(agent_id, direction)
            print(f"[Chaser Ghost] Round {state['round']}, ({gx},{gy})->({px},{py}): {direction} ({r.get('message','?')})")

        except Exception as e:
            print(f"[Chaser Ghost] Error: {e}")
        time.sleep(0.3)

if __name__ == "__main__":
    resp = requests.post(f"{BASE}/api/agent/register", json={"name": "Chaser"})
    data = resp.json()
    print(f"Registered: {data}")
    run(data["agent_id"])
