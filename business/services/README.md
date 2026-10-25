# services — 业务微服务集合

本目录承载业务区的四个后端微服务。它们共享 `vocara/pkg`（公共库）与 `vocara/contracts`（跨边界契约），各自独立编译、独立部署，服务之间通过 HTTP 或事件协作，不直接访问对方存储。

## 服务列表

| 目录 | 服务名 | 端口 | module 名 | 职责 |
|---|---|---|---|---|
| [`admin/`](./admin) | vocara-admin | 8080 | `vocara/admin` | 管理后台 API：组织、权限、线路、路由、监控、授权管理等 |
| [`agentcenter/`](./agentcenter) | vocara-agentcenter | 8081 | `vocara/services/agentcenter` | 座席中心：状态机、技能组、分机、座席会话 |
| [`acd/`](./acd) | vocara-acd | 8082 | `vocara/services/acd` | ACD 智能排队：双 ZSET、Lua 脚本、优先级与老化 |
| [`push/`](./push) | vocara-push | 8084 | `vocara/services/push` | WebSocket 实时推送：NATS + Redis 双消费者 |

各服务的详细说明见其目录内 README。

## 统一结构

```
<service>/
├── cmd/main.go            # 启动入口
├── configs/
│   ├── config.yaml        # 本机标准配置（127.0.0.1 标准端口，可用 CONFIG_PATH 覆盖）
│   └── config.local.yaml  # 开发联调配置（共享联调主机，本地文件，不进版本库）
└── internal/
    ├── handler/           # 接口层
    ├── service/           # 业务逻辑层
    ├── repository/        # 数据访问层
    └── model/             # 数据模型
```

按职责差异：

- `acd/` 另含 `scripts/`：Lua 排队脚本与 `//go:embed` 嵌入包。
- `push/` 另含 `internal/consumer/`：NATS / Redis 事件消费。

## 与核心区的协作

- acd 经 `queue.dequeue` 事件把通话分配给 callctrl，不做 Go 侧同步调用。
- 话单由 callctrl 统一发布 `cdr.created`；事件结构以 `vocara/contracts` 为准。

## 构建

```bash
# 在仓库根，构建全部四个服务
go -C business build ./services/admin/... ./services/agentcenter/... ./services/acd/... ./services/push/...

# 单个服务
go build ./business/services/admin/...

# 交付产包
make build-business      # 产物位于 dist/business
make package-business
```
