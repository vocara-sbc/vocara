# services — Business Microservice Collection

This directory contains the business area's four backend microservices. They share `vocara/pkg` (shared library) and `vocara/contracts` (cross-boundary contracts), and each builds and deploys independently. Services interact through HTTP or events and do not directly access one another's storage.

## Service List

| Directory | Service Name | Port | Module Name | Responsibilities |
|---|---|---|---|---|
| [`admin/`](./admin) | vocara-admin | 8080 | `vocara/admin` | Admin console API: organizations, permissions, trunks, routing, monitoring, authorization management, etc. |
| [`agentcenter/`](./agentcenter) | vocara-agentcenter | 8081 | `vocara/services/agentcenter` | Agent center: state machines, skill groups, extensions, agent sessions |
| [`acd/`](./acd) | vocara-acd | 8082 | `vocara/services/acd` | Intelligent ACD queuing: dual ZSETs, Lua scripts, priority and aging |
| [`push/`](./push) | vocara-push | 8084 | `vocara/services/push` | Real-time WebSocket push: dual NATS + Redis consumers |

See the README in each service's directory for details.

## Common Structure

```
<service>/
├── cmd/main.go            # Startup entry point
├── configs/
│   ├── config.yaml        # Standard local configuration (127.0.0.1 standard ports; override with CONFIG_PATH)
│   └── config.local.yaml  # Development integration configuration (shared integration host; local file excluded from version control)
└── internal/
    ├── handler/           # API layer
    ├── service/           # Business logic layer
    ├── repository/        # Data access layer
    └── model/             # Data models
```

Differences by responsibility:

- `acd/` also includes `scripts/`: Lua queuing scripts and a `//go:embed` embedding package.
- `push/` also includes `internal/consumer/`: NATS / Redis event consumption.

## Interaction with the Core Area

- acd assigns calls to callctrl through `queue.dequeue` events and makes no synchronous Go calls.
- callctrl publishes all call detail records through `cdr.created`; `vocara/contracts` defines the event structure.

## Build

```bash
# Build all four services from the repository root
go -C business build ./services/admin/... ./services/agentcenter/... ./services/acd/... ./services/push/...

# Single service
go build ./business/services/admin/...

# Delivery packages
make build-business      # Artifacts are located in dist/business
make package-business
```
