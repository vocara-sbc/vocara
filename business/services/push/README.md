# Vocara Push Service

The Vocara real-time push service delivers real-time data such as agent state changes, call events, and queue monitoring to frontends over WebSocket.

## Features

- **Multiple WebSocket connection types**: Agent clients, administration clients, and monitoring dashboards
- **Versatile push capabilities**: Specific agents, specific tenants, topic subscriptions, and global broadcasts
- **Subscription mechanism**: Clients can dynamically subscribe to and unsubscribe from topics
- **Multi-node deployment**: Redis routing tables plus Redis Pub/Sub or NATS for cross-node message routing
- **Heartbeat detection**: 30-second ping/pong with automatic disconnection after a 60-second timeout
- **JWT authentication**: Authentication for both WebSocket connections and HTTP APIs
- **Batch push**: Supports batch push requests for greater efficiency
- **Connection statistics**: Real-time statistics for connections, messages, topic subscriptions, and more

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Push Service                           │
├─────────────┬───────────────────┬───────────────────────────┤
│  WS Handler │   Push Handler    │      NATS/Redis Consumer  │
│  (Entry)    │   (HTTP API)      │      (Message Consumer)   │
└──────┬──────┴─────────┬─────────┴──────────────┬────────────┘
       │                │                        │
       ▼                ▼                        ▼
┌─────────────────────────────────────────────────────────────┐
│                     Push Service                            │
│  ┌───────────────┐  ┌─────────────────┐  ┌───────────────┐  │
│  │ Conn Manager  │  │ Sub Service     │  │ Router Service│  │
│  │ (Connections) │  │ (Subscriptions) │  │ (Node Routing)│  │
│  └───────────────┘  └─────────────────┘  └───────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## Project Structure

```
services/push/
├── cmd/
│   └── main.go                 # Service entry point
├── configs/
│   └── config.yaml             # Configuration file
├── internal/
│   ├── model/
│   │   ├── message.go          # Message models (request/response/routing)
│   │   ├── event.go            # Event types, message formats, and data structures
│   │   └── connection.go       # WebSocket connection wrapper
│   ├── service/
│   │   ├── connection_manager.go # Connection manager
│   │   ├── push_service.go     # Push service
│   │   ├── subscription_service.go # Subscription service
│   │   └── router_service.go   # Multi-node routing service
│   ├── handler/
│   │   ├── ws_handler.go       # WebSocket request handler
│   │   └── push_handler.go     # HTTP push API handler
│   └── consumer/
│       ├── nats_consumer.go    # NATS message consumer
│       └── redis_consumer.go   # Redis Pub/Sub message consumer
├── go.mod
└── README.md
```

## Message Protocol

### Server Push Message Format

```json
{
  "event": "agent.status_change",
  "event_id": "evt_20240101120000_abc12345",
  "timestamp": 1700000000000,
  "version": "1.0",
  "data": {}
}
```

### Client Message Format

```json
{
  "action": "subscribe",
  "data": {
    "topics": ["queue:1001", "monitor:tenant:1"]
  }
}
```

### Event Types

#### Agent Events
- `agent.status_change` - Agent state change
- `agent.checkin` - Agent check-in
- `agent.checkout` - Agent check-out
- `agent.busy` - Agent busy
- `agent.ready` - Agent ready

#### Call Events
- `call.ringing` - Incoming call ringing
- `call.answered` - Call answered
- `call.bridged` - Call bridged
- `call.held` - Call placed on hold
- `call.unheld` - Call resumed
- `call.hangup` - Call ended
- `call.transferred` - Call transferred

#### Queue/Monitoring Events
- `queue.update` - Queue status update
- `agent.monitor_update` - Agent monitoring data update
- `stats.update` - Statistics update

#### System Events
- `heartbeat` - Heartbeat
- `system.notice` - System notification
- `error` - Error notification

### Topic Naming Conventions

| Topic Format | Description | Automatic Subscription |
|---------|------|---------|
| `agent:{agentId}` | Agent's personal topic | Automatic for agent clients |
| `tenant:{tenantId}` | Tenant broadcast topic | Automatic for agents/administrators |
| `queue:{queueId}` | Queue monitoring topic | Manual subscription |
| `monitor:tenant:{tenantId}` | Tenant monitoring topic | Automatic for monitoring dashboards |
| `broadcast` | Global broadcast topic | - |

## APIs

### WebSocket APIs

| Path | Description | Authentication |
|------|------|------|
| `GET /ws/agent?token=xxx` | Agent client connection | JWT |
| `GET /ws/admin?token=xxx` | Administration client connection | JWT |
| `GET /ws/monitor?token=xxx` | Monitoring dashboard connection | JWT |

### HTTP Push APIs (Internal Service Calls)

All APIs require the `X-Internal-Token` request header.

| Method | Path | Description |
|------|------|------|
| POST | `/api/v1/push/to-agent` | Push to a specific agent |
| POST | `/api/v1/push/to-tenant` | Push to an entire tenant |
| POST | `/api/v1/push/to-all` | Broadcast to everyone |
| POST | `/api/v1/push/topic` | Push to a specific topic |
| POST | `/api/v1/push/batch` | Batch push |

### Other APIs

| Method | Path | Description |
|------|------|------|
| GET | `/health` | Health check |
| GET | `/api/v1/stats` | Connection statistics |

## Multi-Node Deployment

Multi-node deployment uses Redis routing tables plus Redis Pub/Sub (or NATS):

1. After an agent establishes a WebSocket connection, register a route in Redis:
   - Key: `vocara:ws:route:{agentId}`
   - Value: `{ nodeId, nodeAddr, connId, ... }`
   - TTL: 120 seconds (renewed by heartbeat)

2. When pushing, first query the Redis routing table:
   - Send local connections directly through the channel
   - Forward messages for other nodes through Pub/Sub to `vocara.ws.node:{nodeId}`

3. Delete the route record when the connection closes

4. Renew all local routes with a heartbeat every 30 seconds

## Configuration

Main configuration items (`configs/config.yaml`, standard local configuration). For development integration, use `configs/config.local.yaml` (points to a shared integration host and is not committed to version control):

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
  urls: ""  # Leave empty to use Redis Pub/Sub

log:
  level: info
  format: console
```

Environment variable overrides (prefix `VOCARA_`):
- `VOCARA_SERVER_PORT` - Service port
- `VOCARA_REDIS_ADDRS` - Redis address
- `VOCARA_APP_SECRET_KEY` - Secret key

## Quick Start

### Prerequisites

- Go 1.22+
- Redis 6.0+ (required)
- NATS 2.0+ (optional, for high-throughput scenarios)

### Run

```bash
# Clone the repository
git clone <repo>
cd vocara/services/push

# Install dependencies
go mod download

# Configure
cp configs/config.yaml.example configs/config.yaml
# Edit configs/config.yaml

# Run
go run cmd/main.go

# Or use an environment variable
CONFIG_PATH=/path/to/config.yaml go run cmd/main.go
```

### Build

```bash
go build -o bin/push-service cmd/main.go
```

## Heartbeat Mechanism

- The server sends a ping frame every 54 seconds
- The client must respond with a pong within 60 seconds
- The server closes the connection if no heartbeat is received for more than 60 seconds
- The client can also send a `heartbeat` message as an application-layer heartbeat

## Performance Metrics

- A single node supports tens of thousands of concurrent WebSocket connections
- Single-message push latency < 10ms
- Supports horizontal scaling through multi-node deployment

## License

MIT License
