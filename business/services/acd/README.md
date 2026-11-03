# Vocara ACD Service

The Vocara call center **ACD (Automatic Call Distribution)** queueing service intelligently assigns incoming calls to idle agents.

## Features

- ✅ **Dual ZSET queues**: Active and paused queues support scenarios such as IVR transfer and pending allocation confirmation
- ✅ **Event-driven scheduling**: Events such as enqueue, agent ready, and resume trigger immediate allocation with millisecond-level response times
- ✅ **Priority + aging mechanism**: VIP calls take precedence, priority increases with wait time, and starvation is prevented
- ✅ **Distributed locks**: Only one node allocates calls for a given queue at a time, preventing duplicate allocation
- ✅ **Atomic Lua operations**: All Redis operations use Lua scripts to ensure atomicity
- ✅ **Real-time monitoring**: Query key metrics such as queue length, wait time, and SLA in real time
- ✅ **Scheduled fallback**: Scans every 5 seconds to ensure allocation continues if events are lost

## Technical Architecture

### Core Approach

Uses **Redis dual ZSETs + event-driven scheduling** (Option 4):

```
┌───────────────────────────────────────────────────────────┐
│                        ACD Service                         │
│                                                           │
│  ┌──────────┐   Enqueue   ┌──────────┐   Allocate        │
│  │  Call    │ ──────────▶ │  Active  │ ────────▶ Agent    │
│  │  Ctrl    │             │  Queue   │                    │
│  └──────────┘             └────┬─────┘                    │
│                                │ Pause/Resume             │
│                                ▼                           │
│                           ┌──────────┐                    │
│                           │  Paused  │                    │
│                           │  Queue   │ (IVR/confirmation) │
│                           └──────────┘                    │
└───────────────────────────────────────────────────────────┘
```

### Redis Data Structures

| Key | Type | Description |
|-----|------|------|
| `vocara:acd:queue:active:{queueId}` | ZSET | Active call queue, score = priority * 10^12 + eqTimestamp |
| `vocara:acd:queue:paused:{queueId}` | ZSET | Queue with allocation paused, score = pauseTimestamp |
| `vocara:acd:queue:call:{callId}` | HASH | Call details |
| `vocara:acd:queue:working-ids` | SET | Set of active queue IDs |
| `vocara:acd:queue:stats:{queueId}` | HASH | Queue statistics |
| `vocara:acd:queue:lock:{queueId}` | STRING | Distributed queue allocation lock |

### Priority Algorithm

```
score = priority * 10^12 + eqTimestamp
```

- There are 3 priority levels: VIP (3) > Normal (2) > Low (1)
- Within the same priority, calls that enter the queue earlier (smaller timestamp) rank higher
- Aging is checked on each allocation: priority increases by one level for every `priorityAgingCycle` seconds of waiting
- Priority cannot exceed `maxPriority`

## Project Structure

```
services/acd/
├── go.mod                    # Go module: vocara/service/acd
├── cmd/
│   └── main.go               # Service startup entry point
├── internal/
│   ├── handler/              # HTTP Handler (Hertz)
│   │   ├── acd_handler.go    # ACD API handler
│   │   └── monitor_handler.go # Monitoring API handler
│   ├── service/              # Business logic layer
│   │   ├── acd_service.go    # ACD core service
│   │   ├── scheduler.go      # Event-driven scheduler
│   │   └── monitor_service.go # Monitoring service
│   ├── repository/           # Data access layer
│   │   ├── acd_redis_repo.go # Redis operations (Lua scripts)
│   │   └── stats_repo.go     # Statistics repository
│   └── model/                # Data models
│       ├── call.go           # Queued call model
│       ├── queue.go          # Queue model
│       └── stats.go          # Statistics model
├── scripts/                  # Lua scripts (atomic operations)
│   ├── enqueue.lua           # Enqueue
│   ├── allocate.lua          # Allocation (take queue head)
│   ├── pause.lua             # Pause
│   ├── resume.lua            # Resume
│   └── dequeue.lua           # Dequeue
├── configs/
│   └── config.yaml           # Configuration file
└── README.md
```

## APIs

### Basic Information

- Base URL: `http://<host>:<port>/api/v1/acd`
- Content-Type: `application/json`
- Trace ID: Propagated through the `X-Trace-Id` request header

### 1. Enqueue a Call

**POST** `/enqueue`

Request body:
```json
{
  "callId": "call-001",
  "caller": "13800138000",
  "called": "10086",
  "queueId": 1,
  "skillGroupId": 101,
  "priority": 2,
  "tenantId": 1,
  "customerInfo": "{\"name\":\"Zhang San\",\"level\":\"gold\"}",
  "uui": "Call-associated data"
}
```

Response:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "callId": "call-001",
    "queueId": 1,
    "priority": 2,
    "status": "waiting"
  }
}
```

### 2. Dequeue a Call

**POST** `/dequeue`

Request body:
```json
{
  "callId": "call-001",
  "reason": "answered"   // answered | abandoned | transferred | timeout | error
}
```

### 3. Pause Allocation

**POST** `/pause`

Request body:
```json
{
  "callId": "call-001",
  "reason": "ivr"
}
```

### 4. Resume Allocation

**POST** `/resume`

Request body:
```json
{
  "callId": "call-001"
}
```

### 5. Query Call Information

**GET** `/call/{callId}`

Response:
```json
{
  "code": 0,
  "data": {
    "callId": "call-001",
    "caller": "13800138000",
    "queueId": 1,
    "priority": 2,
    "currentPriority": 3,
    "status": "waiting",
    "eqTime": "2024-01-01T10:00:00Z",
    "tenantId": 1
  }
}
```

### 6. Monitor a Single Queue

**GET** `/monitor/queue/{queueId}`

### 7. Monitor All Queues

**GET** `/monitor/queues?tenantId=1`

Response:
```json
{
  "code": 0,
  "data": [
    {
      "queueId": 1,
      "waitingCount": 15,
      "pausedCount": 3,
      "servingCount": 8,
      "totalToday": 120,
      "answeredToday": 95,
      "abandonedToday": 10,
      "avgWaitTime": 18.5,
      "maxWaitTime": 125,
      "sla": 85.3
    }
  ]
}
```

### 8. Health Check

**GET** `/health`

## Configuration

See `configs/config.yaml` for the complete configuration (standard local configuration). The development integration environment uses `configs/config.local.yaml` (points to a shared integration host and is not committed to version control); load it with `CONFIG_PATH=configs/config.local.yaml`.

| Configuration Key | Type | Default | Description |
|--------|------|--------|------|
| `acd.maxQueueLen` | int | 100 | Maximum queue length; 0 means unlimited |
| `acd.maxPriority` | int | 3 | Maximum priority |
| `acd.priorityAgingCycle` | int | 60 | Priority aging cycle in seconds; 0 disables aging |
| `acd.maxWaitTime` | int | 600 | Maximum wait time in seconds before dequeue on timeout |
| `acd.scheduleInterval` | int | 5 | Scheduled fallback scan interval in seconds |
| `acd.lockTimeout` | int | 10 | Distributed lock timeout in seconds |
| `acd.allocateMaxRetries` | int | 3 | Maximum allocation retries |

## Event-Driven Design

### Subscribed Events

| Event Topic | Publisher | Triggered Action |
|----------|--------|----------|
| `vocara.acd.call.enqueue` | ACD service | Trigger allocation for the corresponding queue |
| `vocara.acd.call.resume` | ACD service | Trigger allocation for the corresponding queue |
| `vocara.agent.ready` | AgentCenter | Trigger allocation for the corresponding queue |

### Published Events

| Event Topic | Trigger |
|----------|----------|
| `vocara.acd.call.enqueue` | A call is successfully enqueued |
| `vocara.acd.call.dequeue` | A call is dequeued for any reason |
| `vocara.acd.call.resume` | Allocation resumes for a call |

Event message format (JSON):
```json
{
  "callId": "call-001",
  "queueId": 1,
  "priority": 2,
  "tenantId": 1,
  "timestamp": 1704067200,
  "traceId": "abc123"
}
```

## Local Development

### Requirements

- Go 1.22+
- Redis 6.0+

### Start the Service

```bash
cd services/acd

# Use go.work mode (recommended)
cd ../..
go work use ./services/acd

# Run
cd services/acd
go run ./cmd
```

### Testing

```bash
# Enqueue test
curl -X POST http://localhost:8081/api/v1/acd/enqueue \
  -H "Content-Type: application/json" \
  -d '{"callId":"test-001","caller":"13800138000","called":"10086","queueId":1,"skillGroupId":101,"priority":2,"tenantId":1}'

# Query queue status
curl http://localhost:8081/api/v1/acd/monitor/queue/1

# Health check
curl http://localhost:8081/health
```

## Design Highlights

### 1. Why Use Dual ZSETs?

- **Active queue**: Calls waiting for allocation, ordered by priority + wait time
- **Paused queue**: Calls with allocation paused (in IVR, allocated and awaiting confirmation, etc.), ordered by pause time
- Separating the queues prevents paused calls from occupying the head of the active queue and reducing allocation efficiency

### 2. Why Use Event-Driven Scheduling with a Scheduled Fallback?

- **Event-driven**: Triggers allocation immediately when calls are enqueued or agents become ready, providing low latency
- **Scheduled fallback**: Prevents queue buildup when events are lost or exceptional scenarios occur
- Together, they provide both low latency and reliability

### 3. Why Use Distributed Locks?

- When multiple ACD service nodes are deployed, several nodes may allocate calls for the same queue simultaneously
- A queue-level distributed lock ensures that only one node allocates calls for that queue at a time
- This prevents duplicate allocation (assigning the same call to multiple agents)

### 4. Aging Mechanism Implementation

- Scores are not updated in batches by a background scheduled task, which has poor performance
- Instead, each allocation checks whether the call at the queue head needs a priority increase
- The score is updated and the queue reordered only when an increase is needed, then `PRIORITY_UPDATED_RETRY` is returned
- The caller retries once to obtain the updated queue head

## Shared Library Dependencies

- `vocara/pkg/config` - Configuration management
- `vocara/pkg/logger` - Logging (supports trace_id propagation)
- `vocara/pkg/errors` - Unified error types
- `vocara/pkg/redis` - Redis client wrapper
- `vocara/pkg/response` - Unified HTTP response format
- `vocara/pkg/database` - Database access
- `vocara/pkg/utils` - Common utility functions

## License

MIT
