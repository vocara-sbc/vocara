# contracts — 跨边界契约

`vocara/contracts` 是业务与核心两侧共享的接口契约模块。它定义跨服务边界的事件与 HTTP 数据结构（DTO），是公开（vocara）与私有（core）两个仓库之间编译期稳定的接口面：业务侧与核心侧都依据本模块开发，编译期互不依赖对方源码。

## 约束

- **零第三方依赖**：仅使用 Go 标准库与结构体 tag，保证任何一侧都可无额外成本引用。
- **只放契约，不放实现**：不包含业务逻辑、持久化模型与传输细节。
- **版本化组织**：HTTP DTO 按 `domain/version` 目录分层（如 `call/v1`），破坏性变更以新版本承载。

## 内容

| 路径 | 内容 |
|---|---|
| `event/allocated.go` | `AllocatedCall`：通话分配事件，主题 `queue.dequeue` |
| `event/cdr.go` | `CDRData`：话单事件，主题 `cdr.created` |
| `call/v1/inbound.go` | callctrl 的 inbound 回调请求体（admit / ring / bridge / hangup） |

事件经统一信封（`vocara/pkg/eventbus`）发布，主题前缀为 `vocara.event.`。

## 引用方

- 业务区：acd 发布分配事件、push 等消费侧按契约解析。
- 核心区：callctrl 订阅 `queue.dequeue`、发布 `cdr.created`，inbound 回调以 `call/v1` 类型接收。
