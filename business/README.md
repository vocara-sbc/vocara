# business — Business Area

The business area contains the call center's business services and admin frontend. This area has its own `go.work` and builds independently **without accessing the core area's source code**. It interacts with the core area through `vocara/contracts` contracts and events / HTTP.

## Contents

```
business/
├── go.work                 # Independent business-area workspace (pkg + contracts + four business services, excluding core)
├── services/
│   ├── admin/              # vocara-admin       :8080  module vocara/admin
│   ├── agentcenter/        # vocara-agentcenter :8081  module vocara/services/agentcenter
│   ├── acd/                # vocara-acd         :8082  module vocara/services/acd
│   └── push/               # vocara-push        :8084  module vocara/services/push
└── web/
    ├── admin/              # Admin console frontend (Vue3 + TS + Vite + Naive UI)
    └── form/               # Dynamic form SDK (Vite library mode)
```

Services use the internal layering `cmd → internal/{handler,service,repository,model}`; acd also includes `scripts/` (Lua scripts and a `//go:embed` embedding package), and push also includes `internal/consumer/`.

Collection index: [`services/README.md`](./services/README.md) (list and builds for the four services), [`web/README.md`](./web/README.md) (admin and form frontend projects).

## Relationships with Other Areas

- Shares `pkg/` (shared library) and `contracts/` (cross-boundary contracts), referenced through `use ../pkg` and `use ../contracts` in go.work.
- Interaction with callctrl: acd issues assignments through `queue.dequeue` events and makes no synchronous Go calls.

## Build

```bash
# Build the business area independently (from the repository root)
go -C business build ./services/admin/... ./services/agentcenter/... ./services/acd/... ./services/push/...

# Frontend
cd business/web/admin && npm run build
```

Delivery builds use `make build-business` / `make package-business` from the repository root.
