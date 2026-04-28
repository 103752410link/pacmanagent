# Pac-Man Agent Arena

AI Agent 对战竞技场 — 吃豆人 + 幽灵，回合制对战。

## 架构

```
main.go                    # 入口: 启动引擎、Agent 注册表、HTTP 服务器
internal/
  game/
    engine.go              # 回合制游戏引擎（核心状态机）
    player.go              # 吃豆人角色，指令驱动移动
    ghost.go               # 幽灵角色，含 BFS 返回路径
    map.go                 # 迷宫地图数据（31行 x 28列）
  agent/
    registry.go            # Agent 注册管理（角色分配）
  server/
    handler.go             # HTTP 路由 + WebSocket 广播
    websocket.go           # WebSocket Hub
static/
  script/index.js          # Canvas 渲染 + WebSocket 客户端
  style/index.css          # 样式
index.html                 # 前端页面
scripts/
  run_agents.py            # Python Agent 脚本（1吃豆人 + 2幽灵）
```

## 回合制流程

```
回合 N 开始
  │
  ▼ [Phase: player_turn] 等待吃豆人提交 2 个移动指令（超时 30s 自动推进）
  ▼ 吃豆人移动 2 步 + 吃豆检测 + 碰撞检测
  ▼ [Phase: ghost_turn]  等待每个幽灵提交 1 个移动指令
  ▼ 幽灵各移动 1 步 + 碰撞检测 + 吃豆检测
  ▼ 回合 N+1
```

- 第一个注册的 Agent 自动成为 **吃豆人**（RolePlayer）
- 后续注册的 Agent 自动成为 **幽灵**（RoleGhost）
- 没有 Agent 注册时，游戏以 AI 模式自动运行
- 幽灵撞到吃豆人（同格）→ Game Over

## 快速开始

### 启动服务器
```bash
go run main.go
# 服务运行在 :8080
# 浏览器: http://localhost:8080
```

### 运行 Python Agent 脚本
```bash
python3 scripts/run_agents.py
# 自动注册 1 吃豆人 + 2 幽灵，开始对战
```

## API 接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/agent/register` | 注册 Agent（首个=吃豆人，其余=幽灵） |
| POST | `/api/agent/:id/action` | 提交移动指令 (direction: left/right/up/down/stay) |
| DELETE | `/api/agent/:id` | 注销 Agent |
| GET | `/api/game/state` | 获取当前游戏状态 |
| GET | `/api/game/maze` | 获取迷宫数据 |
| WS | `/ws` | WebSocket 实时状态广播 |

### 注册示例
```bash
curl -X POST http://localhost:8080/api/agent/register \
  -H "Content-Type: application/json" \
  -d '{"name":"MyBot"}'
# 返回: {"agent_id":"agent-0001","role":"player","color":"yellow"}
```

### 移动示例
```bash
curl -X POST http://localhost:8080/api/agent/agent-0001/action \
  -H "Content-Type: application/json" \
  -d '{"direction":"right"}'
```

## 编写自己的 Agent

### Python 示例
```python
import requests

BASE = "http://localhost:8080"

# 1. 注册
resp = requests.post(f"{BASE}/api/agent/register", json={"name": "MyAgent"})
agent_id = resp.json()["agent_id"]
role = resp.json()["role"]  # "player" 或 "ghost"

# 2. 循环提交指令
while True:
    state = requests.get(f"{BASE}/api/game/state").json()
    if state.get("game_over"):
        break

    if role == "player" and state["phase"] == "player_turn":
        # 吃豆人需要提交 2 个移动
        requests.post(f"{BASE}/api/agent/{agent_id}/action", json={"direction": "right"})
        requests.post(f"{BASE}/api/agent/{agent_id}/action", json={"direction": "down"})

    elif role == "ghost" and state["phase"] == "ghost_turn":
        # 幽灵提交 1 个移动
        requests.post(f"{BASE}/api/agent/{agent_id}/action", json={"direction": "left"})
```

## 方向常量
- Go 端: `0=right, 1=down, 2=left, 3=up`
- 移动: 每步移动到相邻格子
- 隧道: 左右两侧可通过隧道穿越边界

## 幽灵状态
- `1` (Normal): 正常追逐
- `3` (Scared): 被能量球削弱，可被吃豆人吃掉
- `4` (Returning): 被吃后返回重生点 (13,14)

## 碰撞规则
- 精确格碰撞：吃豆人和幽灵必须在同一格
- Normal 幽灵撞到吃豆人 → 吃豆人死亡，Game Over
- Scared 幽灵被吃豆人碰到 → 幽灵变为 Returning，+10 分

## 关卡
- 每关收集所有豆子进入下一关
- 最多 12 关，超过则游戏结束
- 过关时：豆子重置、吃豆人重生、幽灵回到起始位置
