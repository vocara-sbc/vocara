# Vocara Push Service

Vocara 实时推送服务，负责将座席状态变更、通话事件、队列监控等实时数据通过 WebSocket 推送给前端。

## 功能特性

- **多类型 WebSocket 连接**：座席端、管理端、监控大屏
- **丰富的推送能力**：指定座席、指定租户、主题订阅、全局广播
- **订阅机制**：客户端可动态订阅/取消订阅主题
- **多节点部署**：Redis 路由表 + Redis Pub/Sub 或 NATS 实现跨节点消息路由
- **心跳检测**：30 秒 ping/pong，60 秒超时自动断开
- **JWT 鉴权**：WebSocket 连接和 HTTP 接口均支持鉴权
- **批量推送**：支持批量推送请求，提升效率
- **连接统计**：实时统计连接数、消息数、主题订阅数等

## 架构设计

```
┌─────────────────────────────────────────────────────────────┐
│                      Push Service                           │
├─────────────┬───────────────────┬───────────────────────────┤
│  WS Handler │   Push Handler    │      NATS/Redis Consumer  │
│  (入口)     │   (HTTP 接口)     │      (消息消费)            │
└──────┬──────┴─────────┬─────────┴──────────────┬────────────┘
       │                │                        │
       ▼                ▼                        ▼
┌─────────────────────────────────────────────────────────────┐
│                     Push Service                            │
│  ┌───────────────┐  ┌─────────────────┐  ┌───────────────┐  │
│  │ Conn Manager  │  │ Sub Service     │  │ Router Service│  │
│  │ (连接管理)    │  │ (订阅管理)      │  │ (多节点路由)  │  │
│  └───────────────┘  └─────────────────┘  └───────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## 项目结构

```
services/push/
├── cmd/
│   └── main.go                 # 服务入口
├── configs/
│   └── config.yaml             # 配置文件
├── internal/
│   ├── model/
│   │   ├── message.go          # 消息模型（请求/响应/路由）
│   │   ├── event.go            # 事件类型、消息格式、数据结构
│   │   └── connection.go       # WebSocket 连接封装
│   ├── service/
│   │   ├── connection_manager.go # 连接管理器
│   │   ├── push_service.go     # 推送服务
│   │   ├── subscription_service.go # 订阅服务
│   │   └── router_service.go   # 多节点路由服务
│   ├── handler/
│   │   ├── ws_handler.go       # WebSocket 请求处理器
│   │   └── push_handler.go     # HTTP 推送接口处理器
│   └── consumer/
│       ├── nats_consumer.go    # NATS 消息消费者
│       └── redis_consumer.go   # Redis Pub/Sub 消息消费者
├── go.mod
└── README.md
```

## 消息协议

### 服务端推送消息格式

```json
{
  "event": "agent.status_change",
  "event_id": "evt_20240101120000_abc12345",
  "timestamp": 1700000000000,
  "version": "1.0",
  "data": {}
}
```

### 客户端消息格式

```json
{
  "action": "subscribe",
  "data": {
    "topics": ["queue:1001", "monitor:tenant:1"]
  }
}
```

### 事件类型

#### 座席相关
- `agent.status_change` - 座席状态变更
- `agent.checkin` - 座席签入
- `agent.checkout` - 座席签出
- `agent.busy` - 座席示忙
- `agent.ready` - 座席示闲

#### 通话相关
- `call.ringing` - 来电振铃
- `call.answered` - 通话接听
- `call.bridged` - 通话桥接
- `call.held` - 通话保持
- `call.unheld` - 通话恢复
- `call.hangup` - 通话结束
- `call.transferred` - 通话转接

#### 队列/监控相关
- `queue.update` - 队列状态更新
- `agent.monitor_update` - 座席监控数据更新
- `stats.update` - 统计数据更新

#### 系统相关
- `heartbeat` - 心跳
- `system.notice` - 系统通知
- `error` - 错误通知

### 主题命名规范

| 主题格式 | 说明 | 自动订阅 |
|---------|------|---------|
| `agent:{agentId}` | 座席个人主题 | 座席端自动 |
| `tenant:{tenantId}` | 租户广播主题 | 座席/管理员自动 |
| `queue:{queueId}` | 队列监控主题 | 手动订阅 |
| `monitor:tenant:{tenantId}` | 租户监控主题 | 监控大屏自动 |
| `broadcast` | 全局广播主题 | - |

## API 接口

### WebSocket 接口

| 路径 | 说明 | 鉴权 |
|------|------|------|
| `GET /ws/agent?token=xxx` | 座席端连接 | JWT |
| `GET /ws/admin?token=xxx` | 管理端连接 | JWT |
| `GET /ws/monitor?token=xxx` | 监控大屏连接 | JWT |

### HTTP 推送接口（内部服务调用）

所有接口需携带 `X-Internal-Token` 请求头。

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/push/to-agent` | 推送给指定座席 |
| POST | `/api/v1/push/to-tenant` | 推送给整个租户 |
| POST | `/api/v1/push/to-all` | 广播给所有人 |
| POST | `/api/v1/push/topic` | 推送到指定主题 |
| POST | `/api/v1/push/batch` | 批量推送 |

### 其他接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/health` | 健康检查 |
| GET | `/api/v1/stats` | 连接统计信息 |

## 多节点部署方案

使用 Redis 路由表 + Redis Pub/Sub（或 NATS）实现多节点：

1. 座席建立 WebSocket 连接后，在 Redis 注册路由：
   - Key: `vocara:ws:route:{agentId}`
   - Value: `{ nodeId, nodeAddr, connId, ... }`
   - TTL: 120 秒（心跳续期）

2. 推送时先查 Redis 路由表：
   - 本地连接直接通过 channel 发送
   - 其他节点通过 Pub/Sub 转发到 `vocara.ws.node:{nodeId}`

3. 连接断开时删除路由记录

4. 每 30 秒心跳续期所有本地路由

## 配置说明

主要配置项（`configs/config.yaml`，本机标准配置）。开发联调时使用 `configs/config.local.yaml`（指向共享联调主机，不进版本库）：

```bash
CONFIG_PATH=configs/config.local.yaml go run cmd/main.go
```

```yaml
app:
  name: vocara-push
  secret_key: "your-secret-key"

server:
  host: 0.0.0.0
  port: 8083

redis:
  addrs: "127.0.0.1:6379"
  password: ""
  db: 0

nats:
  urls: ""  # 留空则使用 Redis Pub/Sub

log:
  level: info
  format: console
```

环境变量覆盖（前缀 `VOCARA_`）：
- `VOCARA_SERVER_PORT` - 服务端口
- `VOCARA_REDIS_ADDRS` - Redis 地址
- `VOCARA_APP_SECRET_KEY` - 密钥

## 快速开始

### 前置依赖

- Go 1.22+
- Redis 6.0+（必须）
- NATS 2.0+（可选，用于高吞吐量场景）

### 运行

```bash
# 克隆仓库
git clone <repo>
cd vocara/services/push

# 安装依赖
go mod download

# 配置
cp configs/config.yaml.example configs/config.yaml
# 编辑 configs/config.yaml

# 运行
go run cmd/main.go

# 或使用环境变量
CONFIG_PATH=/path/to/config.yaml go run cmd/main.go
```

### 构建

```bash
go build -o bin/push-service cmd/main.go
```

## 心跳机制

- 服务端每 54 秒发送一次 ping 帧
- 客户端需在 60 秒内回复 pong
- 超过 60 秒未收到心跳，服务端主动断开连接
- 客户端也可发送 `heartbeat` 消息进行应用层心跳

## 性能指标

- 单节点支持万级并发 WebSocket 连接
- 单条消息推送延迟 < 10ms
- 支持水平扩展（多节点部署）

## 许可证

MIT License
