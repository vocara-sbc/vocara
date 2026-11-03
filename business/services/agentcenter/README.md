# Vocara Agent Center (Agent Center Service)

The agent center is a core service of the Vocara call center. It manages the full agent lifecycle, including login, check-in/check-out, state management, skill groups, and extension management.

## Features

- **Full agent lifecycle management**: Login/logout, check-in/check-out, and heartbeat keepalive
- **12-state agent state machine**: Strict state transition validation
- **Skill group management**: Multiple skill groups per agent, with skill levels
- **Extension management**: Extension registration, status queries, and online detection
- **Idle agent management**: Redis ZSET-based ordering by longest idle time
- **Agent allocation**: Distributed locks ensure concurrency safety and prevent over-allocation
- **Heartbeat expiration detection**: Periodically scans for expired agents and automatically sets them to OFFLINE
- **JWT authentication**: Token authentication for agent APIs
- **Internal service authentication**: Dedicated token authentication for internal services such as ACD

## Agent State Machine

```
OFFLINE(0) → CHECKIN(1) → CHECKED(2) → READY(3) ←→ BUSY(9)
                                      ↑         ↓
                                      │    WAIT_SERVER(4)
                                      │         ↓
                               AFTER_WORK(8) ← RINGING(5)
                                      ↑         ↓
                                      └── CALLING(7) ← OUTBOUNDING(6)
                                     (LISTENING=71 / THREE_WAY=72 substates)
```

### State Descriptions

| State | Enum Value | Description | Assignable by ACD |
|------|--------|------|-------------|
| OFFLINE | 0 | Offline | ❌ |
| CHECKIN | 1 | Checking in | ❌ |
| CHECKED | 2 | Checked in (not ready) | ❌ |
| READY | 3 | Idle and ready | ✅ |
| WAIT_SERVER | 4 | Waiting for service (awaiting answer after assignment) | ❌ |
| RINGING | 5 | Ringing | ❌ |
| OUTBOUNDING | 6 | Outbound call in progress | ❌ |
| CALLING | 7 | On a call | ❌ |
| LISTENING | 71 | Listen-in active (substate) | ❌ |
| THREE_WAY | 72 | Three-way call (substate) | ❌ |
| AFTER_WORK | 8 | After-call work | ❌ |
| BUSY | 9 | Busy | ❌ |

### State Transition Rules

| Current State | Allowed Transitions |
|----------|----------|
| OFFLINE | CHECKIN |
| CHECKIN | CHECKED, OFFLINE |
| CHECKED | READY, OFFLINE |
| READY | BUSY, WAIT_SERVER, OUTBOUNDING, OFFLINE |
| BUSY | READY, OFFLINE |
| WAIT_SERVER | RINGING, READY |
| RINGING | CALLING, AFTER_WORK, READY |
| OUTBOUNDING | CALLING, READY |
| CALLING | AFTER_WORK, LISTENING, THREE_WAY, READY |
| LISTENING | CALLING, AFTER_WORK |
| THREE_WAY | CALLING, AFTER_WORK |
| AFTER_WORK | READY, BUSY, OFFLINE |

## Redis Data Structures

```bash
# Agent session (real-time state)
vocara:agent:session:{tenantId}:{agentId}   HASH
  Fields: agentId, agentName, status, dnNumber, tenantId,
          skillGroups, lastReadyTime, currentCallId,
          lastActiveTime, checkinTime, ip

# Online agent list
vocara:agent:online:{tenantId}               SET
  Members: agentId

# Idle agents (by skill group)
vocara:agent:idle:{skillGroupId}             ZSET
  score = lastReadyTimestamp  (longest idle time first)
  member = agentId

# Agent-to-skill-group mapping
vocara:agent:skills:{agentId}                SET
  Members: skillGroupId

# Agent distributed lock
vocara:agent:lock:{agentId}                  STRING
  TTL = 5s (temporary lock during allocation)
```

## Database Tables

| Table | Description |
|------|------|
| tb_agent | Agent information |
| tb_organization | Organizations |
| tb_skill_group | Skill groups |
| tb_agent_skill_group | Agent-to-skill-group relationships |
| tb_dn | Extensions |
| tb_online_agent | Online agent records |
| tb_agent_service_record | Agent service records |

## Project Structure

```
services/agentcenter/
├── go.mod
├── cmd/
│   └── main.go                 # Service entry point
├── internal/
│   ├── handler/                # HTTP handler layer
│   │   ├── agent_handler.go    # Agent APIs
│   │   ├── dn_handler.go       # Extension APIs
│   │   ├── skill_handler.go    # Skill group APIs
│   │   └── manage_handler.go   # Internal management APIs (for ACD)
│   ├── service/                # Business logic layer
│   │   ├── agent_service.go    # Agent service (state machine core)
│   │   ├── session_service.go  # Session management (Redis)
│   │   ├── dn_service.go       # Extension service
│   │   └── skill_service.go    # Skill group service
│   ├── repository/             # Data access layer
│   │   ├── agent_repo.go       # Agent data access
│   │   ├── dn_repo.go          # Extension data access
│   │   └── skill_repo.go       # Skill group data access
│   └── model/                  # Data models
│       ├── agent.go            # Agent model & DTO
│       ├── dn.go               # Extension model & DTO
│       ├── skill.go            # Skill group model & DTO
│       └── status.go           # State enums & state machine
├── configs/
│   └── config.yaml             # Configuration file
└── README.md
```

## APIs

### Agent APIs

| Method | Path | Description | Authentication |
|------|------|------|------|
| POST | /api/v1/agent/login | Agent login | No |
| POST | /api/v1/agent/heartbeat/{agentId} | Heartbeat keepalive | No |
| POST | /api/v1/agent/logout | Agent logout | JWT |
| POST | /api/v1/agent/checkin | Agent check-in | JWT |
| POST | /api/v1/agent/checkout | Agent check-out | JWT |
| POST | /api/v1/agent/busy | Set busy | JWT |
| POST | /api/v1/agent/ready | Set ready | JWT |
| GET | /api/v1/agent/info | Get agent information | JWT |
| GET | /api/v1/agent/status | Get agent status | JWT |

### Extension APIs

| Method | Path | Description | Authentication |
|------|------|------|------|
| GET | /api/v1/dn/list | Extension list | JWT |
| GET | /api/v1/dn/{dnId} | Extension detail | JWT |
| POST | /api/v1/dn/checkonline | Batch extension online check | JWT |

### Skill Group APIs

| Method | Path | Description | Authentication |
|------|------|------|------|
| GET | /api/v1/skill/list | Skill group list | JWT |
| GET | /api/v1/skill/{id} | Skill group detail | JWT |
| GET | /api/v1/skill/{id}/agents | Agents in a skill group | JWT |

### Internal Management APIs (for ACD)

Authentication requires the `X-Internal-Token` request header.

| Method | Path | Description |
|------|------|------|
| POST | /api/v1/internal/agent/try-acquire | Try to allocate an idle agent |
| POST | /api/v1/internal/agent/ring | Notify an agent to ring |
| POST | /api/v1/internal/agent/resume-ready | Reset an agent to idle when allocation fails |
| POST | /api/v1/internal/agent/update-status | Update agent status |

## Core Design

### Agent Allocation Flow

1. ACD calls `/api/v1/internal/agent/try-acquire` with `skill_group_id`
2. Pop the agent with the lowest score from the ZSET `vocara:agent:idle:{skillGroupId}` (longest idle time first)
3. Retrieve the agent session and verify that the agent is actually in the READY state
4. Acquire a distributed lock (`vocara:agent:lock:{agentId}`, 5-second TTL)
5. Validate the state again and transition to WAIT_SERVER
6. Release the lock and return the agent information
7. If allocation fails (agent offline or invalid state), automatically retry with the next agent, up to 3 attempts

### Heartbeat Keepalive Mechanism

- The agent session key has a TTL (120 seconds by default)
- The agent client calls the heartbeat API periodically (recommended every 30 seconds)
- The heartbeat API refreshes the session TTL and lastActiveTime
- A background scanner (every 30 seconds by default) scans all online sets
- Agents with expired sessions are automatically cleaned up and set to OFFLINE

### Idle Agent ZSET

- Each skill group has one ZSET: `vocara:agent:idle:{skillGroupId}`
- score = lastReadyTimestamp (timestamp when the agent entered the READY state)
- `ZPOPMIN` pops the agent that became ready earliest (longest idle time first)
- Agents are removed from all related ZSETs when they leave the READY state

## Configuration

The configuration file is located at `configs/config.yaml`. It contains the standard local configuration (middleware points to standard ports on `127.0.0.1`) and supports specifying a path through the `CONFIG_PATH` environment variable. The development integration configuration is `configs/config.local.yaml` (points to a shared integration host and is not committed to version control). Specify it explicitly at startup:

```bash
CONFIG_PATH=configs/config.local.yaml go run cmd/main.go
```

```yaml
server:
  host: "0.0.0.0"
  port: 8080

mysql:
  host: "127.0.0.1"
  port: 3306
  user: "root"
  password: "123456"
  database: "vocara_agent"

redis:
  host: "127.0.0.1"
  port: 6379
  db: 0

jwt:
  secret: "your-jwt-secret"
  expire_hour: 24

agent:
  session_ttl: 120       # Session expiration time (seconds)
  scan_interval: 30000   # Expiration scan interval (milliseconds)

internal:
  token: "internal-token"  # Internal service authentication token
```

## Quick Start

### Requirements

- Go 1.22+
- MySQL 5.7+ / 8.0+
- Redis 6.0+

### Start the Service

```bash
# Enter the project directory
cd services/agentcenter

# Install dependencies
go mod tidy

# Start the service
go run cmd/main.go

# Or specify a configuration file
CONFIG_PATH=/path/to/config.yaml go run cmd/main.go
```

### Build

```bash
# Build the binary
go build -o bin/agent-center cmd/main.go

# Run
./bin/agent-center
```

### Health Check

```bash
curl http://localhost:8080/health
# {"status":"ok","service":"agent-center"}
```

## Error Codes

| Error Code | Description |
|--------|------|
| 0 | Success |
| 40000 | Invalid parameter |
| 40100 | Unauthorized |
| 40101 | Invalid token |
| 40102 | Token expired |
| 40103 | Token missing |
| 40300 | Access forbidden |
| 40400 | Resource not found |
| 50000 | System error |
| 50002 | Database error |
| 50003 | Redis error |
| 10000 | Agent not found |
| 10001 | Agent not checked in |
| 10002 | Agent already checked in |
| 10003 | Agent busy |
| 10004 | Agent offline |
| 10005 | Invalid agent state |
| 10006 | Incorrect password |
| 10007 | Agent already online |
| 10008 | Agent not ready |
| 10009 | Invalid state transition |
| 10010 | No idle agents |
| 10011 | Extension not found |
| 10012 | Extension unavailable |
| 10013 | Failed to acquire distributed lock |

## Dependencies

- [Hertz](https://github.com/cloudwego/hertz) - HTTP framework
- [GORM](https://gorm.io/) - ORM framework
- [go-redis](https://github.com/redis/go-redis) - Redis client
- [golang-jwt](https://github.com/golang-jwt/jwt) - JWT library
- [zap](https://github.com/uber-go/zap) - Logging library
- [viper](https://github.com/spf13/viper) - Configuration management
- [bcrypt](https://golang.org/x/crypto/bcrypt) - Password hashing

## License

Vocara
