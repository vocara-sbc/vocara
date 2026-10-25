# business — 业务区

业务区承载呼叫中心的业务服务与管理前端。本区有独立的 `go.work`，可在**不接触核心区源码**的前提下独立编译；与核心区之间通过 `vocara/contracts` 契约和事件 / HTTP 协作。

## 内容

```
business/
├── go.work                 # 业务区独立工作区（pkg + contracts + 业务四服务，不含 core）
├── services/
│   ├── admin/              # vocara-admin       :8080  module vocara/admin
│   ├── agentcenter/        # vocara-agentcenter :8081  module vocara/services/agentcenter
│   ├── acd/                # vocara-acd         :8082  module vocara/services/acd
│   └── push/               # vocara-push        :8084  module vocara/services/push
└── web/
    ├── admin/              # 管理后台前端（Vue3 + TS + Vite + Naive UI）
    └── form/               # 动态表单 SDK（Vite 库模式）
```

服务内部分层为 `cmd → internal/{handler,service,repository,model}`；acd 另含 `scripts/`（Lua 脚本与 `//go:embed` 嵌入包），push 另含 `internal/consumer/`。

集合索引：[`services/README.md`](./services/README.md)（四服务列表与构建）、[`web/README.md`](./web/README.md)（admin 与 form 前端工程）。

## 与其它区的关系

- 共享 `pkg/`（公共库）与 `contracts/`（跨边界契约），经 go.work 的 `use ../pkg`、`use ../contracts` 引用。
- 与 callctrl 的协作：acd 经 `queue.dequeue` 事件下发分配，不做 Go 侧同步调用。

## 构建

```bash
# 业务区独立构建（在仓库根）
go -C business build ./services/admin/... ./services/agentcenter/... ./services/acd/... ./services/push/...

# 前端
cd business/web/admin && npm run build
```

交付构建走仓库根 `make build-business` / `make package-business`。
