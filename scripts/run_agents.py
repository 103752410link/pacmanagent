#!/usr/bin/env python3
"""Run all agents: 1 Pac-Man + 2 Ghosts - single script, no race conditions."""
import requests, time, threading, sys

BASE = "http://localhost:8080"
DIRS = ["right", "down", "left", "up"]

def get_state():
    return requests.get(f"{BASE}/api/game/state").json()

def get_maze():
    return requests.get(f"{BASE}/api/game/maze").json()["maze"]

def submit(agent_id, direction):
    r = requests.post(f"{BASE}/api/agent/{agent_id}/action", json={"direction": direction})
    if r.status_code != 200:
        print(f"[DEBUG] submit({agent_id}, {direction}) -> {r.status_code}: {r.text}")
    return r.json()

def bfs_nearest(sx, sy, targets, maze):
    """BFS to nearest target. Returns first direction."""
    if not targets:
        return "stay"
    tset = set(targets)
    if (sx, sy) in tset:
        return "stay"
    visited = set()
    queue = [(sx, sy, [])]
    visited.add((sx, sy))
    while queue:
        x, y, path = queue.pop(0)
        if (x, y) in tset:
            return path[0] if path else "stay"
        for i, (dx, dy) in enumerate([(1,0),(0,1),(-1,0),(0,-1)]):
            nx, ny = x + dx, y + dy
            if 0 <= nx < 28 and 0 <= ny < len(maze) and maze[ny][nx] < 1 and (nx, ny) not in visited:
                visited.add((nx, ny))
                queue.append((nx, ny, path + [DIRS[i]]))
    return "stay"

# ============ Pac-Man Bot ============
def run_pacman(agent_id):
    print(f"[Pac-Man] {agent_id} started")
    sys.stdout.flush()
    while True:
        s = get_state()
        if s.get("game_over"):
            print(f"[Pac-Man] Game over. Score: {s['score']}")
            break
        if s["phase"] != "player_turn":
            time.sleep(0.2)
            continue
        if s.get("player_moves_submitted", 0) >= 2:
            time.sleep(0.2)
            continue

        maze = get_maze()
        px, py = s["player"]["coord_x"], s["player"]["coord_y"]
        ghosts = s.get("ghosts") or []
        beans = s.get("beans") or []
        bean_set = set((b["X"], b["Y"]) for b in beans if b["X"] != px or b["Y"] != py)

        # Check nearby ghosts
        ghost_near = any(
            abs(g["coord_x"] - px) + abs(g["coord_y"] - py) <= 5
            for g in ghosts if g.get("status", 1) == 1
        )

        moves = []
        cx, cy = px, py
        for _ in range(2):
            if ghost_near:
                best_d, best_dist = "stay", -1
                for d, (dx, dy) in enumerate([(1,0),(0,1),(-1,0),(0,-1)]):
                    nx, ny = cx + dx, cy + dy
                    if 0 <= nx < 28 and 0 <= ny < len(maze) and maze[ny][nx] < 1:
                        min_g = min(abs(g["coord_x"]-nx) + abs(g["coord_y"]-ny) for g in ghosts)
                        if min_g > best_dist:
                            best_dist = min_g
                            best_d = DIRS[d]
                moves.append(best_d)
            else:
                bean_set = set((b["X"], b["Y"]) for b in beans if b["X"] != cx or b["Y"] != cy)
                moves.append(bfs_nearest(cx, cy, list(bean_set), maze))

            cx += {"right":1,"left":-1,"up":0,"down":0}.get(moves[-1], 0)
            cy += {"right":0,"left":0,"up":-1,"down":1}.get(moves[-1], 0)
            if cx < 0: cx = 27
            if cx > 27: cx = 0

        # Submit BOTH moves back-to-back (no polling between)
        for m in moves:
            submit(agent_id, m)
        print(f"[Pac-Man] R{s['round']} S{s['score']}: {moves[0]}+{moves[1]}")
        time.sleep(0.15)

# ============ Ghost Bot ============
def run_ghost(agent_id, strategy):
    name = "Chaser" if strategy == "chase" else "Ambush"
    print(f"[{name} Ghost] {agent_id} started")
    while True:
        s = get_state()
        if s.get("game_over"):
            print(f"[{name}] Game over. Score: {s['score']}")
            break
        if s["phase"] != "ghost_turn":
            time.sleep(0.2)
            continue

        maze = get_maze()
        px, py = s["player"]["coord_x"], s["player"]["coord_y"]

        my_ghost = None
        for g in s.get("ghosts") or []:
            if g["id"] == agent_id:
                my_ghost = g
                break
        if not my_ghost:
            time.sleep(0.2)
            continue
        gx, gy = my_ghost["coord_x"], my_ghost["coord_y"]

        if strategy == "chase":
            tx, ty = px, py
        else:
            deltas = {0:(4,0), 1:(0,4), 2:(-4,0), 3:(0,-4)}
            dx, dy = deltas.get(s["player"]["orientation"], (0, 0))
            tx = max(0, min(27, px + dx))
            ty = max(0, min(len(maze)-1, py + dy))

        direction = bfs_nearest(gx, gy, [(tx, ty)], maze)
        r = submit(agent_id, direction)
        if "accepted" in r.get("message", "") or "round complete" in r.get("message", ""):
            print(f"[{name}] R{s['round']} ({gx},{gy})->({tx},{ty}): {direction}")
        time.sleep(0.15)

# ============ Main ============
def main():
    print("=== Registering Agents ===")
    p = requests.post(f"{BASE}/api/agent/register", json={"name": "PacBot"}).json()
    print(f"Pac-Man: {p['agent_id']} ({p['role']})")

    g1 = requests.post(f"{BASE}/api/agent/register", json={"name": "Chaser"}).json()
    print(f"Ghost 1: {g1['agent_id']} ({g1['role']})")

    g2 = requests.post(f"{BASE}/api/agent/register", json={"name": "Ambush"}).json()
    print(f"Ghost 2: {g2['agent_id']} ({g2['role']})")

    print("\n=== Starting Agents ===")
    t1 = threading.Thread(target=run_pacman, args=(p["agent_id"],), daemon=True)
    t2 = threading.Thread(target=run_ghost, args=(g1["agent_id"], "chase"), daemon=True)
    t3 = threading.Thread(target=run_ghost, args=(g2["agent_id"], "ambush"), daemon=True)

    t1.start()
    t2.start()
    t3.start()

    # Monitor
    while True:
        time.sleep(3)
        s = get_state()
        alive = s["player"]["alive"]
        print(f"\n--- Round {s['round']}, Score {s['score']}, Alive {alive}, Beans {s['beans_remaining']} ---")
        if not alive or s.get("game_over"):
            print(f"Game Over! Final Score: {s['score']}")
            break

if __name__ == "__main__":
    main()
