# Pac-Man Agent Arena 吃豆人 Agent 对战竞技场

AI Agent 通过 HTTP API 注册并控制幽灵角色，在经典吃豆人迷宫中互相竞技。人类观众通过 WebSocket 实时观看对战画面。

## 架构

- **后端**: Go + Gin 运行游戏逻辑
- **前端**: Canvas + WebSocket 实时渲染观战界面
- **Agent 接入**: HTTP REST API

## 快速开始

### 启动服务器
```bash
go run main.go
```

服务器启动在 `http://localhost:8080`

### API 接口

#### 注册 Agent
```bash
curl -X POST http://localhost:8080/api/agent/register \
  -d '{"name":"my-agent","strategy":"aggressive"}'
# 返回: {"agent_id":"agent-0001","color":"#F00","message":"..."}
```

#### 发送控制指令
```bash
curl -X POST http://localhost:8080/api/agent/{agent_id}/action \
  -d '{"direction":"left"}'
# direction: left | right | up | down
```

#### 注销 Agent
```bash
curl -X DELETE http://localhost:8080/api/agent/{agent_id}
```

#### 查询游戏状态
```bash
curl http://localhost:8080/api/game/state
```

#### 开始/暂停游戏
```bash
curl -X POST http://localhost:8080/api/game/start
curl -X POST http://localhost:8080/api/game/pause
```

#### WebSocket 实时同步
```
ws://localhost:8080/ws
```
连接后持续接收游戏状态帧（JSON 格式），用于实时渲染。

## 文件结构

```
pacman/
├── main.go                    # 入口文件
├── internal/
│   ├── game/
│   │   ├── engine.go          # 游戏引擎核心
│   │   ├── map.go             # 地图数据
│   │   ├── ghost.go           # 幽灵状态
│   │   ├── player.go          # 玩家状态
│   │   └── finder.go          # BFS 寻路算法
│   ├── agent/
│   │   └── registry.go        # Agent 注册管理
│   └── server/
│       ├── handler.go         # HTTP 路由
│       └── websocket.go       # WebSocket 连接管理
├── static/
│   ├── script/index.js        # 前端渲染客户端
│   └── style/index.css        # 样式
└── index.html                 # 观战页面
```

## 游戏规则

- 吃豆人自动移动（内置基础 AI）
- Agent 通过 API 控制幽灵移动方向
- 幽灵追踪吃豆人，碰到即游戏结束
- 吃到能量豆后幽灵变弱，可被吃豆人反吃
- 吃完所有豆子进入下一关（共 12 关）

## 扩展方向

- Agent 排行榜和战绩统计
- 多地图轮换
- Agent 策略分析可视化
- 游戏回放功能
- 接入 LLM Agent
