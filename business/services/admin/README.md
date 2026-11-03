# Vocara Admin Service

The Vocara call center administration service provides agent management, queue management, skill group management, inbound reception configuration, call records, statistical reports, and other features.

## Tech Stack

- **Framework**: Hertz (CloudWeGo)
- **Database**: MySQL (GORM v2)
- **Cache**: Redis (go-redis v9)
- **Authentication**: JWT
- **Logging**: Zap + Lumberjack
- **Configuration**: Viper

## Project Structure

```
services/admin/
├── cmd/
│   └── main.go              # Service entry point
├── configs/
│   └── config.yaml          # Configuration file
├── internal/
│   ├── handler/             # HTTP handler layer
│   │   ├── auth_handler.go       # Authentication
│   │   ├── agent_handler.go      # Agent management
│   │   ├── agent_group_handler.go # Agent group management
│   │   ├── skill_group_handler.go # Skill group management
│   │   ├── dn_handler.go         # Extension management
│   │   ├── queue_handler.go      # Queue management
│   │   ├── admit_handler.go      # Inbound reception
│   │   ├── record_handler.go     # Call records
│   │   ├── line_handler.go       # Line resources
│   │   ├── stats_handler.go      # Statistical reports
│   │   └── system_handler.go     # System management
│   ├── service/             # Business logic layer
│   │   ├── agent_service.go
│   │   ├── agent_group_service.go
│   │   ├── skill_group_service.go
│   │   ├── dn_service.go
│   │   ├── queue_service.go
│   │   ├── admit_service.go
│   │   ├── record_service.go
│   │   ├── line_service.go
│   │   ├── stats_service.go
│   │   ├── system_service.go
│   │   └── tenant_service.go
│   ├── repository/          # Data access layer
│   │   ├── base_repo.go
│   │   ├── agent_repo.go
│   │   ├── agent_group_repo.go
│   │   ├── skill_group_repo.go
│   │   ├── dn_repo.go
│   │   ├── queue_repo.go
│   │   ├── admit_repo.go
│   │   ├── record_repo.go
│   │   ├── line_repo.go
│   │   └── system_repo.go
│   └── model/               # Data models
│       ├── models.go
│       └── stats.go
├── go.mod
└── README.md
```

## APIs

All APIs use the prefix `/api/v1/admin/`.

### Auth APIs

| Method | Path | Description | Authentication |
|------|------|------|------|
| POST | /auth/login | Administrator login | No |
| POST | /auth/logout | Logout | Yes |
| POST | /auth/change_password | Change password | Yes |
| GET  | /auth/userinfo | Get user information | Yes |

### Agent Management

| Method | Path | Description |
|------|------|------|
| GET | /agents/page | Paginated query |
| GET | /agents/list | List query |
| GET | /agents/:id | Detail |
| POST | /agents/save | Create |
| POST | /agents/update | Update |
| DELETE | /agents/:id | Delete |
| POST | /agents/import | Batch import |
| GET | /agents/export | Export |
| POST | /agents/:id/skill | Update skills |
| POST | /agents/:id/bind_dn | Bind extension |
| POST | /agents/:id/reset_password | Reset password |
| POST | /agents/:id/enable | Enable |
| POST | /agents/:id/disable | Disable |

### Agent Group Management

| Method | Path | Description |
|------|------|------|
| GET | /agent_groups/page | Paginated query |
| GET | /agent_groups/list | List query |
| GET | /agent_groups/:id | Detail |
| POST | /agent_groups/save | Create |
| POST | /agent_groups/update | Update |
| DELETE | /agent_groups/:id | Delete |
| GET | /agent_groups/:id/members | Member list |
| POST | /agent_groups/:id/members/add | Add member |
| POST | /agent_groups/:id/members/remove | Remove member |
| POST | /agent_groups/:id/leader | Set group leader |

### Skill Group Management

| Method | Path | Description |
|------|------|------|
| GET | /skill_groups/page | Paginated query |
| GET | /skill_groups/list | List query |
| GET | /skill_groups/:id | Detail |
| POST | /skill_groups/save | Create |
| POST | /skill_groups/update | Update |
| DELETE | /skill_groups/:id | Delete |
| GET | /skill_groups/:id/agents | Agent list |
| POST | /skill_groups/:id/agents | Set agents |

### Extension Management

| Method | Path | Description |
|------|------|------|
| GET | /dns/page | Paginated query |
| GET | /dns/list | List query |
| GET | /dns/:id | Detail |
| POST | /dns/save | Create |
| POST | /dns/update | Update |
| DELETE | /dns/:id | Delete |
| POST | /dns/status_check | Status check |
| POST | /dns/batch_import | Batch import |
| GET | /dns/available | Available extension list |
| POST | /dns/:id/bind_agent | Bind agent |
| POST | /dns/:id/unbind_agent | Unbind agent |

### Queue Management

| Method | Path | Description |
|------|------|------|
| GET | /queues/page | Paginated query |
| GET | /queues/list | List query |
| GET | /queues/:id | Detail |
| POST | /queues/save | Create |
| POST | /queues/update | Update |
| DELETE | /queues/:id | Delete |
| GET | /queues/monitor/list | Real-time monitoring |
| GET | /queues/:id/agents | Queue agents |

### Inbound Reception

| Method | Path | Description |
|------|------|------|
| GET | /admits/page | Paginated query |
| GET | /admits/list | List query |
| GET | /admits/:id | Detail |
| POST | /admits/save | Create |
| POST | /admits/update | Update |
| DELETE | /admits/:id | Delete |
| GET | /admits/holiday/page | Paginated holiday query |
| GET | /admits/holiday/list | Holiday list |
| GET | /admits/holiday/:id | Holiday detail |
| POST | /admits/holiday/save | Create holiday |
| POST | /admits/holiday/update | Update holiday |
| DELETE | /admits/holiday/:id | Delete holiday |

### Call Records

| Method | Path | Description |
|------|------|------|
| GET | /records/inbound/page | Paginated inbound call records |
| GET | /records/outbound/page | Paginated outbound call records |
| GET | /records/:id | Call detail |
| GET | /records/export | Export |
| GET | /records/today | Today's statistics |
| GET | /records/uncalled | Missed call list |
| GET | /records/:id/recording | Recording information |

### Line Resources

| Method | Path | Description |
|------|------|------|
| GET | /lines/page | Paginated query |
| GET | /lines/list | List query |
| GET | /lines/:id | Detail |
| POST | /lines/save | Create |
| POST | /lines/update | Update |
| DELETE | /lines/:id | Delete |
| GET | /lines/status/info | Status statistics |

### Statistical Reports

| Method | Path | Description |
|------|------|------|
| GET | /stats/agent | Agent call traffic statistics |
| GET | /stats/queue | Queue statistics |
| GET | /stats/line | Line statistics |
| GET | /stats/trend | Trend statistics |
| GET | /stats/realtime | Real-time statistics |

### System Management

| Method | Path | Description |
|------|------|------|
| GET | /system/param/list | Parameter configuration list |
| POST | /system/param/save | Save configuration |
| DELETE | /system/param/:id | Delete configuration |
| GET | /system/operation_log/page | Paginated operation log query |
| GET | /system/fs/page | Paginated FS instance query |
| GET | /system/fs/list | FS instance list |
| GET | /system/fs/:id | FS instance detail |
| POST | /system/fs/save | Create FS instance |
| POST | /system/fs/update | Update FS instance |
| DELETE | /system/fs/:id | Delete FS instance |
| GET | /system/tenant/page | Paginated tenant query |
| GET | /system/tenant/list | Tenant list |
| GET | /system/tenant/:id | Tenant detail |
| POST | /system/tenant/save | Create tenant |
| POST | /system/tenant/update | Update tenant |
| DELETE | /system/tenant/:id | Delete tenant |

## Middleware

Requests pass through the following middleware chain:

1. **TraceID** - Generates or propagates the trace ID
2. **Recovery** - Recovers from panics
3. **CORS** - Handles cross-origin requests
4. **RequestLogger** - Request logging
5. **JWT** - Authentication (skipped for the login API)

## Multi-tenancy Design

- All business tables have a `tenant_id` field
- Administrators obtain a default tenant after login
- Query operations automatically apply tenant isolation conditions
- System-level data (FS instances and global configuration) is not restricted by tenant

## Start

```bash
# Install dependencies
go mod download

# Build
go build -o admin ./cmd/main.go

# Run
./admin
```

## Configuration

See `configs/config.yaml` (standard local configuration). Environment variable overrides are supported with the prefix `VOCARA_`. The development integration environment uses `configs/config.local.yaml` (points to a shared integration host and is not committed to version control); load it with `CONFIG_PATH=configs/config.local.yaml`.
