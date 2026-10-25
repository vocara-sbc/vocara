# vocara 呼叫中心系统 — FreeSWITCH ESL 交互层设计

## 5.1 定位

FreeSWITCH 经 Event Socket Library（ESL）暴露 TCP 控制端口（默认 8021）。Vocara 在
核心仓库 `esl`（module `vocara/core/esl`）提供自研 Go 实现，承担与 FreeSWITCH 之间的
命令发送与事件接收。呼叫控制服务 vocara-callctrl 经此库以 Inbound 客户端方式连接
FreeSWITCH，完成呼入、呼出、排队振铃、桥接、录音、监听耳语等编排。

```
┌──────────────┐    ESL (TCP 8021)    ┌──────────────┐
│ vocara-callctrl│ ◄────────────────► │ FreeSWITCH   │
│ (Inbound 客户端)│   命令 / 事件       │ (媒体与协议)  │
└──────────────┘                     └──────────────┘
```

库同时实现 Inbound 客户端与 Outbound 服务端；业务侧只使用 Inbound 客户端。

## 5.2 包结构

核心仓库 `esl` 根下为统一入口 `esl.go`，其余按职责划分为六个子包：

| 子包 | 职责 |
|---|---|
| `command` | 命令模型：`ApiCommand`（api/bgapi）与 `AppCommand`（sendmsg 执行应用） |
| `event` | 事件模型 `EslEvent`、统一事件解析、头部常量、挂机原因 |
| `inbound` | Inbound 客户端：连接、认证、订阅、命令收发、自动重连 |
| `outbound` | Outbound 服务端（接受 FreeSWITCH 主动连入） |
| `pool` | 多连接连接池及其配置 |
| `transport` | 底层帧传输与编解码 |

## 5.3 Inbound 客户端

### 5.3.1 连接与收发模型

`inbound.Client` 是业务侧使用的客户端，模型如下：

- 建立 TCP 连接后发送 `auth` 完成认证；
- 单独的读循环协程是唯一从连接读取数据的位置；
- 命令经请求通道送读写循环，按请求/响应方式返回，避免读写竞争；
- 事件解析后经独立事件通道分发到已注册的监听器。

ESL 在同一连接上的 `command/reply`、`api/response` 不携带请求标识，回复按序到达，
因此命令在单条连接上严格串行：同一时刻只允许一条命令在飞，待其回复、超时或连接
断开后才发送下一条。

### 5.3.2 客户端能力

| 方法 | 说明 |
|---|---|
| `Connect` / `Disconnect` | 建立（含认证）/ 断开连接 |
| `Subscribe` | 以 `event plain` 订阅事件 |
| `AddEventFilter` / `DeleteEventFilter` | 增删事件过滤器 |
| `AddEventListener` / `RemoveEventListener` | 注册/移除事件监听器 |
| `Api` | 发送同步 `api` 命令，返回响应体 |
| `BgApi` | 发送异步 `bgapi` 命令，返回 Job UUID；结果经 `BACKGROUND_JOB` 事件回传 |
| `ExecuteApp` | 经 sendmsg 在指定通道上执行应用 |
| `ExecuteApiCmd` / `ExecuteAppCmd` | 以 `command` 包的命令对象发送命令 |

默认时间参数：拨号超时 5s，**命令超时 10s**，重连等待上限 30s。

### 5.3.3 自动重连

连接断开时客户端进入重连循环，按指数退避重连：等待 1s、2s、4s…… 上限 30s。
重连成功后恢复订阅的事件集合与事件过滤器，随后继续读循环。重连过程不发布专门事件。

启动时若 FreeSWITCH 不可达，客户端不阻断进程，在后台保持重连。

## 5.4 命令模型

`command` 包以接口表达两类命令：

- `ApiCommand`：对应 `api`（同步）与 `bgapi`（异步，`IsAsync` 判定），`Build`
  生成不含前缀的命令串；
- `AppCommand`：对应 sendmsg `execute`，携带通道 UUID、应用名、应用参数与循环次数。

命令按 FreeSWITCH 命令族组织为具体实现：

| 命令族 | 文件 | 覆盖命令 |
|---|---|---|
| API | `api_originate.go`、`api_uuid_bridge.go`、`api_uuid_kill.go`、`api_uuid_getvar.go`、`api_uuid_setvar.go`、`api_uuid_record.go`、`api_uuid_send_dtmf.go`、`api_misc.go` | originate、uuid_bridge、uuid_kill、uuid_getvar、uuid_setvar、uuid_record、uuid_send_dtmf 等 |
| APP | `app_answer.go`、`app_hangup.go`、`app_playback.go`、`app_bridge.go`、`app_transfer.go`、`app_set.go`、`app_conference.go`、`app_detect_speech.go` | answer、hangup、playback、bridge、transfer、set、conference、detect_speech 等 |

同步命令直接取响应；`bgapi originate` 等异步命令先取得 Job UUID，再由 `BACKGROUND_JOB`
事件按 Job UUID 关联最终结果。

## 5.5 事件模型与统一解析

`event.EslEvent` 字段：

| 字段 | 含义 |
|---|---|
| `Name` | 事件名，如 `CHANNEL_HANGUP` |
| `Subclass` | 自定义事件子类（`Name == CUSTOM` 时有效） |
| `Header` | 事件头部键值对（键名大小写敏感，保持原样） |
| `Body` | 事件体 |
| `Timestamp` / `Sequence` | 由 `Event-Date-Timestamp`、`Event-Sequence` 解析 |
| `ContentType` | 帧内容类型 |

事件解析统一走 `ParseEventContent`：`ParseEvent` 从 reader 解析完整帧，
`ParseEventContent` 解析帧体内的头部与 body。Inbound 客户端与任何调用方都复用
此入口，不另写解析逻辑。

常用头部以常量定义于 `headers.go`，前缀 `Header`，如 `HeaderUniqueID`、
`HeaderCallerIDNumber`、`HeaderDestinationNumber`、`HeaderAnswerTime`、
`HeaderHangupCause`、`HeaderBridgeUniqueID`、`HeaderDTMFDigit`、`HeaderJobUUID`。
通道变量在事件头部以 `variable_` 前缀出现，经 `GetVariable` 读取。

## 5.6 事件订阅与分发

vocara-callctrl 订阅的事件集合：

| 事件 | 用途 |
|---|---|
| `CHANNEL_CREATE` / `CHANNEL_DESTROY` | 感知呼入、清理通道资源 |
| `CHANNEL_ANSWER` | 通道接听 |
| `CHANNEL_HANGUP` / `CHANNEL_HANGUP_COMPLETE` | 挂机处理、CDR 生成 |
| `CHANNEL_BRIDGE` / `CHANNEL_UNBRIDGE` | 桥接与解桥 |
| `CHANNEL_PROGRESS` / `CHANNEL_PROGRESS_MEDIA` | 早期媒体 |
| `CHANNEL_STATE` | 通道状态变化 |
| `CHANNEL_EXECUTE` / `CHANNEL_EXECUTE_COMPLETE` | 应用执行结果 |
| `RECORD_START` / `RECORD_STOP` | 录音状态 |
| `DTMF` | 按键 |
| `BACKGROUND_JOB` | bgapi 结果回调 |

事件分发采用有界缓冲通道：解析后的事件进入事件通道，由单个分发循环顺序出队，
分发时向每个已注册监听器各启动一个 goroutine 并发回调（监听器 panic 自动恢复），
并发只发生在向监听器 fan-out 阶段。事件通道满时该事件被静默丢弃，不阻塞读循环。

## 5.7 连接池

`pool` 包提供多连接池，配置项 `pool.Config` 及默认值：

| 参数 | 默认值 |
|---|---|
| `MinIdle` / `MaxIdle` | 2 / 10 |
| `MaxActive` | 50 |
| `MaxWait` | 5s |
| `IdleTimeout` | 5m |
| `HealthCheckInterval` | 30s |
| `ConnectTimeout` / `CommandTimeout` | 3s / 10s |

业务侧当前以单 Inbound 客户端连接受控节点；连接池供需要并发命令连接的场景使用。

## 5.8 Outbound 服务端

`outbound` 子包提供 Outbound 服务端能力：FreeSWITCH 作为客户端主动连入服务端，
由服务端在该连接上控制通道。业务侧采用 Inbound 模式，Outbound 服务端为库自带能力。

## 5.9 callctrl 呼叫编排

### 5.9.1 单受控节点

vocara-callctrl 持有一个 Inbound 客户端，连接配置节 `esl` 指定的单一 FreeSWITCH
端点（座席侧节点 FS-Agent，实例标识 `callctrl.fs_instance`）。两个媒体节点的拨号
方案均由 admin 经 mod_xml_curl 的 dialplan binding 按节点（`node=agent` /
`node=sbc`）与 context 实时下发，规则不落节点磁盘。FS-SBC 承担接入侧转发，
不经过 vocara-callctrl 的 ESL 编排。

```
运营商/客户 ──SIP──► FS-SBC ──bridge──► FS-Agent :5080 (park)
                                         ▲ ESL :8021
                                  vocara-callctrl
座席话机/浏览器 ──SIP/WebRTC──────────► FS-Agent
```

### 5.9.2 呼入编排

1. 客户呼入经 FS-SBC 按动态下发的 public 规则 bridge 到 FS-Agent，通道应答后
   park，通道变量 `vocara_did` 记录接入号；
2. vocara-callctrl 接收通道事件，以呼入路由缓存（admin 动态下发为主、配置静态
   路由为回落）匹配接入号到队列或技能组；
3. 进入 ACD 排队，订阅分配事件 `queue.dequeue`；
4. 分配成功后将客户腿 `uuid_transfer` 到拨号方案扩展 `vocara_agent_<分机>`；
5. 拨号方案以 bridge 应用原生创建座席腿 `user/<分机>`，座席接听即在桥中；
6. `CHANNEL_HANGUP_COMPLETE` 统一处理挂机与 CDR。

### 5.9.3 呼出编排

座席侧外呼在 FS-Agent 上发起：按动态下发的 default 规则 bridge 到 FS-SBC，再由
SBC 的出局规则经网关送往运营商，出局规则受 License 状态控制（未授权时替换为
阻断段）；管理接口触发的外呼经 `originate`（`bgapi` 异步、`BACKGROUND_JOB`
回收结果）发起。

### 5.9.4 动态目录与动态拨号方案的边界

FreeSWITCH 的 mod_xml_curl 向 admin 请求两类资源：

| Binding | admin 端点 | 用途 |
|---|---|---|
| directory | `POST /api/v1/fs/directory` | SIP 注册凭据动态化，以 `tb_dn` 为凭据源 |
| dialplan | `POST /api/v1/fs/dialplan` | 按节点（`node=agent` / `node=sbc`）与 context 实时下发拨号方案，规则不落盘 |

ACD 的异步分配语义由 vocara-callctrl 在事件与命令层完成，不依赖 dialplan binding；
座席腿由 dialplan 的 bridge 应用创建。

### 5.9.5 通道变量约定

| 变量 | 含义 |
|---|---|
| `vocara_did` | 呼入接入号 |
| `vocara_call_id` | 通话标识 |
| `vocara_tenant_id` | 租户标识 |
| `vocara_agent_id` | 座席标识 |
| `vocara_queue_id` | 队列标识 |
| `vocara_call_type` | 通话类型（inbound / outbound / internal） |
| `vocara_role` | 通道角色（customer / agent） |
