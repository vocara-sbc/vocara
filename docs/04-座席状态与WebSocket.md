# 04-座席状态与 WebSocket

座席状态由 **vocara-agentcenter（8081）** 统一持有与流转：状态机定义在座席服务内，
会话实时态存 Redis，当前在线记录落 MySQL。状态变化只发布事件，不直接面向浏览器；
**vocara-push（8084）** 作为唯一 WebSocket 网关，消费业务事件并路由到座席、管理端与
监控大屏连接，并承担多节点路由、主题订阅与座席命令网关职责。

## 4.1 座席状态机

### 4.1.1 状态枚举

座席状态共 12 个状态值，定义在 `agentcenter/internal/model/status.go`：

| 值 | 常量 | 名称 | 含义 |
|---|---|---|---|
| 0 | StatusOffline | OFFLINE | 离线，无会话 |
| 1 | StatusCheckin | CHECKIN | 签入中，会话已建立、签入校验进行中 |
| 2 | StatusChecked | CHECKED | 已签入未就绪，不能被分配 |
| 3 | StatusReady | READY | 空闲就绪，可被 ACD 分配 |
| 4 | StatusWaitServer | WAIT_SERVER | 已被 ACD 选中，等待座席侧应答 |
| 5 | StatusRinging | RINGING | 座席分机振铃中 |
| 6 | StatusOutbounding | OUTBOUNDING | 外呼呼出中，等待被叫应答 |
| 7 | StatusCalling | CALLING | 通话中 |
| 71 | StatusListening | LISTENING | 监听中，CALLING 的子状态 |
| 72 | StatusThreeWay | THREE_WAY | 三方通话中，CALLING 的子状态 |
| 8 | StatusAfterWork | AFTER_WORK | 话后处理，不参与分配 |
| 9 | StatusBusy | BUSY | 示忙，不参与分配 |

LISTENING、THREE_WAY 以 70 段编码表示其依附于 CALLING：进入子状态不改变通话归属，
子状态退出后回到 CALLING。

### 4.1.2 状态转换图

```
┌─────────┐   ┌─────────┐   ┌─────────┐   ┌───────┐
│ OFFLINE │──►│ CHECKIN │──►│ CHECKED │──►│ READY │◄──────────────┐
└─────────┘   └─────────┘   └─────────┘   └──┬────┘               │
                                              │                    │
                       ┌──────────────────────┼──────────────┐     │
                       ▼                      ▼              ▼     │
                    ┌──────┐           ┌─────────────┐ ┌──────────┴──┐
                    │ BUSY │           │ WAIT_SERVER │ │ OUTBOUNDING │
                    └──┬───┘           └──────┬──────┘ └──────┬──────┘
                       ▲                      ▼               │
                       │                ┌─────────┐           │
                       │                │ RINGING │           │
                       │                └────┬────┘           │
                       │                     ▼                ▼
                       │              ┌──────────────────────────┐
                       │              │         CALLING          │
                       │              │ LISTENING / THREE_WAY 子态│
                       │              └────────────┬─────────────┘
                       │                           ▼
                       │                    ┌────────────┐
                       └────────────────────│ AFTER_WORK │
                                            └────────────┘
```

### 4.1.3 合法转换（CanTransition）

所有状态变更必须经 `CanTransition` 校验，相同状态互转直接放行：

| 当前状态 | 允许目标状态 |
|---|---|
| OFFLINE | CHECKIN |
| CHECKIN | CHECKED、OFFLINE |
| CHECKED | READY、OFFLINE |
| READY | BUSY、WAIT_SERVER、OUTBOUNDING、OFFLINE |
| BUSY | READY、OFFLINE |
| WAIT_SERVER | RINGING、READY |
| RINGING | CALLING、AFTER_WORK、READY |
| OUTBOUNDING | CALLING、READY |
| CALLING | AFTER_WORK、LISTENING、THREE_WAY、READY |
| LISTENING | CALLING、AFTER_WORK |
| THREE_WAY | CALLING、AFTER_WORK |
| AFTER_WORK | READY、BUSY、OFFLINE |

呼入主链：READY → WAIT_SERVER → RINGING → CALLING → AFTER_WORK → READY。
呼出主链：READY → OUTBOUNDING → CALLING → AFTER_WORK → READY。

### 4.1.4 状态判定方法

| 方法 | 语义 |
|---|---|
| IsIdle() | 仅 READY 为真，是 ACD 可分配的唯一判据 |
| IsOnline() | OFFLINE、CHECKIN 以外均为在线 |
| IsInCall() | WAIT_SERVER、RINGING、OUTBOUNDING、CALLING、LISTENING、THREE_WAY |
| IsSubStatus() | LISTENING、THREE_WAY |
| BaseStatus() | 两个子状态归一为 CALLING，其余原样返回 |

## 4.2 座席会话存储（Redis）

### 4.2.1 Key 总览

| Key | 类型 | 含义 |
|---|---|---|
| `vocara:agent:session:{tenantId}:{agentId}` | HASH | 座席会话，TTL 120s |
| `vocara:agent:online:{tenantId}` | SET | 租户当前在线座席集合 |
| `vocara:agent:idle:{skillGroupId}` | ZSET | 技能组空闲就绪座席，score 为进入 READY 的时间戳 |
| `vocara:agent:skills:{agentId}` | SET | 座席所属技能组集合 |
| `vocara:agent:lock:{agentId}` | STRING | 座席分配锁，TTL 5s |

### 4.2.2 会话 HASH

会话 HASH 保存座席实时工作态，共 11 个字段：

| 字段 | 内容 |
|---|---|
| agentId | 座席 ID |
| agentName | 座席名称 |
| tenantId | 租户 ID |
| status | 当前状态数值 |
| dnNumber | 绑定分机号 |
| skillGroups | 技能组 ID 逗号拼接 |
| currentCallId | 当前通话 ID |
| lastReadyTime | 最近进入 READY 的时间戳 |
| lastActiveTime | 最近活跃时间戳 |
| checkinTime | 签入时间戳 |
| ip | 接入 IP |

每次写入会话都会刷新 120s TTL。读会话以 HASH 是否存在作为座席在线的判据。

### 4.2.3 空闲 ZSET 与最长空闲优先

座席进入 READY 时 `ZADD vocara:agent:idle:{skillGroupId} {lastReadyTs} {agentId}`，
离开 READY 时从其所属全部技能组的空闲集合移除。

ACD 获取座席用 `ZPOPMIN`：score 最小即进入 READY 最早、等待分配最久的座席优先弹出，
天然实现最长空闲优先，弹出动作同时把座席移出空闲集合，配合二次校验防止超卖。

跨技能组的全局就绪总数由 `GetGlobalIdleCount` 计算：SCAN 全部空闲 ZSET
（跳过 `vocara:agent:idle:union:` 前缀的临时键），用 `ZUnionStore` 合并求并集后
`ZCARD` 去重计数，避免同一多技能座席被重复统计。

### 4.2.4 座席锁

分配座席时加锁防并发：`SET vocara:agent:lock:{agentId} {唯一value} NX EX 5`，
value 形如 `lock-{纳秒时间戳}-{agentID}`；释放锁通过 Lua 比对 value 后删除，
只有持锁者能解锁。

### 4.2.5 会话保活与过期扫描

- **保活**：座席 WebSocket 连接在线期间，push 的座席命令网关每 30s 调用 agentcenter
  内部接口刷新会话，会话 HASH 持续续期 120s。
- **过期扫描**：agentcenter 内置扫描器每 30s 执行一次，SCAN 全部
  `vocara:agent:online:*`，以 pipeline EXISTS 找出在线集合中会话 HASH 已不存在的座席，
  依次清理其空闲 ZSET 成员、技能 SET、在线 SET 成员并删除会话。
- **断连不立即改状态**：WebSocket 断开只停止保活，座席状态保持原值；会话在 120s 内
  未被新连接续期才由扫描器清理。这样网络抖动或同座席重连不会产生状态抖动。

扫描间隔（30s）、会话 TTL（120s）为代码默认值。

## 4.3 在线记录（MySQL tb_online_agent）

`tb_online_agent` 记录**当前在线座席**，不是历史表：

- 签入成功按租户 + 座席号 Upsert 一条记录；
- 状态变更时更新 status、status_name；
- 保活到达时更新心跳时间；
- 签出或会话过期清理时删除记录。

座席的主数据为 `tb_agent`，技能组为 `tb_skill_group`，座席与技能组的关联为
`tb_agent_skill`（含 skill_level 技能等级 1-10，默认 5；联合唯一索引
(skill_group_id, agent_id)）。

## 4.4 座席操作链

### 4.4.1 签入

签入由 `AgentService.Checkin` 完成，主流程：

1. 校验座席存在且属于当前租户；
2. 校验绑定分机存在且状态可用；
3. 查重：已有会话则拒绝重复签入；
4. 校验座席技能组归属；
5. 建立 CHECKIN 会话，加入在线集合；
6. 写入座席技能 SET，绑定座席与分机；
7. 状态推进到 CHECKED；
8. 写服务记录，Upsert `tb_online_agent`；
9. 发布 `agent.checkin` 事件。

### 4.4.2 就绪与示忙

座席从 CHECKED 经就绪操作进入 READY 时，先做就绪上限校验
（`GetGlobalIdleCount` 结合 License 在线座席上限），通过后批量加入所属技能组的
空闲 ZSET。离开 READY（示忙、被分配、外呼、签出）时从全部空闲 ZSET 移除。

示忙进入 BUSY，BUSY 与 READY 之间可双向切换。

### 4.4.3 状态变更（TransitionStatus）

`TransitionStatus` → `doTransition` 的统一处理：

1. `CanTransition` 校验源状态到目标状态的合法性；
2. READY 相关的空闲集合增删与就绪上限校验；
3. `SetSession` 更新会话 HASH 并续期；
4. 异步更新 MySQL 座席状态；
5. 发布 `agent.status_change` 事件，载荷为 AgentEventData：

```json
{
  "agent_id": 1001,
  "tenant_id": 1,
  "agent_name": "座席姓名",
  "status": "RINGING",
  "status_text": "振铃中",
  "prev_status": "READY",
  "extension": "1001",
  "skill_groups": [11, 12]
}
```

`UpdateStatus` 支持 force 与 reason 参数，供呼叫控制侧在强制纠偏场景使用。

### 4.4.4 座席获取（TryAcquireAgent）

ACD 分配通话时经内部接口调用 `TryAcquireAgent`，最多重试 3 次：

1. `ZPOPMIN` 从技能组空闲 ZSET 弹出最长空闲座席；
2. 读取会话并校验状态、租户、技能组归属；
3. 获取座席锁；
4. 锁内二次校验座席仍为 READY（防止两节点同时弹出同一座席）；
5. 座席状态推进到 WAIT_SERVER，返回座席信息；
6. 释放座席锁。

### 4.4.5 振铃与退回就绪

- `Ring`：WAIT_SERVER → RINGING，绑定 currentCallId；
- `ResumeReady`：分配失败（座席无应答、呼叫控制侧分配异常）时把座席退回 READY，
  重新加入空闲 ZSET，等待下一轮分配。

### 4.4.6 座席事件

| 事件 | 常量 | 发布时机 |
|---|---|---|
| `agent.checkin` | EventAgentCheckIn | 签入完成 |
| `agent.checkout` | EventAgentCheckOut | 签出完成 |
| `agent.status_change` | EventAgentStatusChange | 任何成功的状态变更 |

事件统一走事件总线信封，主题为 `vocara.event.{事件类型}`。

## 4.5 WebSocket 接入

### 4.5.1 端点

push 服务在 `/ws` 组下提供三个 WebSocket 端点：

| 端点 | 连接类型 | 用途 |
|---|---|---|
| `/ws/agent` | agent | 座席工作台、电话条 |
| `/ws/admin` | admin | 管理后台在线状态 |
| `/ws/monitor` | monitor | 监控大屏 |

### 4.5.2 鉴权

握手时支持 query 参数 token 或 Authorization Bearer 头携带 JWT，从 claims 解析
user_id、username、agent_num（座席工号）、tenant_id。座席连接的用户标识取座席工号，
与通话事件中的 agent_id 对齐。

### 4.5.3 连接生命周期

```
鉴权 → AddConnection → autoSubscribe → RegisterRoute → StartWritePump
     →（座席）AgentGateway.OnConnect → sendWelcome → readPump
     → cleanupConnection
```

- 欢迎帧 event 为 `connected`，携带 conn_id、user_type、user_id、tenant_id、
  server_time、subscriptions；
- cleanupConnection 依次执行 UnsubscribeAll、UnregisterRoute、RemoveConnection、
  AgentGateway.OnDisconnect；
- 座席连接 OnConnect 失败时下发 session/error 帧，等待 100ms 后关闭连接，保证错误帧
  写出。

### 4.5.4 心跳与读超时

- 服务端设 pongWait 60s，期内未收到任何消息（含心跳/Pong）则关闭连接；
- 客户端可发 heartbeat / pong 上行帧保活，服务端回应 pong；
- 业务层保活与会话 TTL 续期由座席命令网关的 30s 周期负责，与 WebSocket 层 60s
  读超时相互独立。

## 4.6 主题订阅

### 4.6.1 主题规范

| 主题 | 形态 | 内容 |
|---|---|---|
| 座席个人 | `agent:{agentId}` | 该座席本人事件 |
| 租户 | `tenant:{tenantId}` | 租户内管理向事件 |
| 队列 | `queue:{queueId}` | 队列状态事件 |
| 监控 | `monitor:tenant:{tenantId}` | 监控大屏事件 |
| 广播 | `broadcast` | 全局广播 |

主题合法性由 `IsValidTopic` 校验：broadcast 或四类前缀（前缀后必须非空）。

### 4.6.2 自动订阅

连接注册后按连接类型自动订阅，无需客户端声明：

| 连接类型 | 自动订阅 |
|---|---|
| agent | agent:{本人}、tenant:{租户}、broadcast |
| admin | tenant:{租户}、broadcast |
| monitor | monitor:tenant:{租户}、broadcast |

订阅关系由 SubscriptionService 以内存映射维护（主题 → 连接集合），连接断开时
UnsubscribeAll 清空。客户端也可经 subscribe/unsubscribe 帧动态增减订阅。

### 4.6.3 上行消息

| action | 含义 |
|---|---|
| subscribe / unsubscribe | 增减主题订阅，data 含 topics |
| heartbeat / pong | 连接保活 |
| ready / busy | 座席示闲 / 示忙 |
| signout | 座席签出 |

## 4.7 多节点路由

### 4.7.1 路由表

每条连接在所属 push 节点注册路由：

- Key：`vocara:ws:route:{agentId}`，值为 RouteInfo JSON
  （agent_id、tenant_id、node_id、node_addr、conn_id、server_time）；
- TTL 120s，节点 heartbeatLoop 每 30s 批量 Expire 为本节点全部路由续期；
- 注销路由时比对 conn_id，防止旧连接的清理误删同名座席的新连接路由。

### 4.7.2 节点通道与推送模式

| 通道 | 用途 |
|---|---|
| `vocara.ws.node.{nodeId}` | 跨节点定向投递：先查路由表，目标在其他节点时发往该节点通道 |
| `vocara.ws.local` | 本节点内广播：广播类推送经此通道在所有节点间扩散 |

推送支持两种底层模式：

- **nats 模式**：NATS 配置非空时启用，节点通道走 NWS 主题
  （发布前缀 `vocara.ws.push.`，节点间前缀 `vocara.ws.node.`）；
- **redis 模式**：默认模式，复用同一套通道名走 Redis Pub/Sub。
NATS 初始化失败时仅告警并回落到 redis 模式。

### 4.7.3 推送方法

| 方法 | 路由方式 |
|---|---|
| PushToAgent | 查路由表：同节点直投，异节点经节点通道转发 |
| PushToTenant | 投递给租户内全部 admin / monitor 连接 |
| PushToTopic | 投递给指定主题的全部订阅连接 |
| PushToAll | 经 local 通道广播到全部节点、全部连接 |

push 同时对内部服务开放 HTTP 接口（`/api/v1/push/to-agent`、`/to-tenant`、
`/to-all`、`/topic`、`/batch`），经内部服务鉴权。

## 4.8 业务事件消费者

### 4.8.1 事件来源

BusinessConsumer 与事件总线的两种实现一一对应：NATS 订阅版与 Redis 订阅版，
订阅全部 `vocara.event.*` 业务主题。

### 4.8.2 路由规则

消费者按事件类型前缀路由到 WebSocket：

| 前缀 | 事件 | 路由 |
|---|---|---|
| `agent.` | 座席状态、签入签出、示闲示忙 | PushToAgent 本人 + 租户内 admin/monitor |
| `call.` | 振铃、接通、桥接、保持、挂断、转接 | PushToAgent 本人 + 租户内 admin/monitor |
| `queue.` | 入队、出队、放弃、超时、位置 | PushToTopic(queue:{id}) + 租户内 admin |
| `system.notice` | 系统通知 | 按通知范围广播或定向 |

座席通话类事件只推送给当事座席本人，管理侧同步看到的是经租户主题分发的副本。

## 4.9 座席命令网关 AgentGateway

push 服务内的 AgentGateway 是座席连接与 agentcenter 之间的命令层：

### 4.9.1 连接即签入与踢旧

- 座席连接完成 WS 注册后调用 agentcenter 内部接口 EstablishSession，该接口幂等：
  会话已存在时等同续期，不存在时按连接绑定的分机建立会话；
- 检测到同一座席已有其他连接在线时，向旧连接下发 kick 帧（reason：duplicate_login）
  并关闭旧连接，同一座席同一时刻只保留一条有效连接；
- 踢旧连接时跳过当前连接自身。

### 4.9.2 保活

- 保活周期 30s，每轮调用内部刷新会话接口，单次请求 8s 超时；
- 连续失败上限 3 次：达到上限下发 session/error 帧并关闭连接；
- OnDisconnect 只停止保活循环，不调用任何状态变更接口；状态以会话 TTL 过期 +
  agentcenter 扫描为准。

### 4.9.3 命令处理

- ready / busy：透传座席示闲、示忙；
- signout：调用内部接口删除会话，回 ack 帧后关闭连接。

所有关闭操作均先发帧、等待 100ms 再关连接，保证对端收到最后一帧。

## 4.10 WS 消息协议

### 4.10.1 下行信封

```json
{
  "event": "agent.status_change",
  "event_id": "evt_20260929143015_a1b2c3d4",
  "timestamp": 1780120215000,
  "version": "1.0",
  "data": { }
}
```

- event_id 由 `evt_` + 时间格式 `20060102150405` + 8 位随机串组成；
- 跨节点转发时用 NewMessageWithID 保持 event_id 与 timestamp 一致；
- timestamp 为毫秒级 Unix 时间。

### 4.10.2 系统帧

| event | 方向 | 用途 |
|---|---|---|
| connected | 下行 | 欢迎帧，连接建立成功 |
| pong | 下行 | 心跳应答，含 server_time、load |
| kick | 下行 | 连接被取代，如 duplicate_login |
| error | 下行 | 错误通知 |
| heartbeat / pong | 上行 | 连接层保活 |

### 4.10.3 业务数据结构

| 结构 | 对应事件 | 主要字段 |
|---|---|---|
| AgentStatusData | agent.* | agent_id、status、prev_status、extension、skill_groups |
| CallEventData | call.* | call_uuid、direction、主被叫、应答/结束时间、hangup_cause、duration |
| CallQueueEventData | queue.enqueue/dequeue/abandoned/timeout | call_id、queue_id、position、reason、wait_time |
| QueueStatusData | queue.update | waiting_num、idle/busy_agent_num、avg/max_wait_time、sla_level |
| AgentMonitorData | agent.monitor_update | status、status_duration、today_calls、today_talk_time |
| SystemNoticeData | system.notice | type、title、content、level |

## 4.11 内部接口

agentcenter 在 `/api/v1/internal`（InternalAuth 中间件）对 push 等内部服务开放：

| 方法与路径 | 用途 |
|---|---|
| POST /agent/session | 建立座席会话（幂等） |
| POST /agent/session/refresh | 刷新会话 TTL |
| POST /agent/session/delete | 删除会话 |
| POST /agent/try-acquire | 弹出并锁定空闲座席 |
| POST /agent/ring | 座席进入振铃 |
| POST /agent/resume-ready | 座席退回就绪 |
| POST /agent/update-status | 强制更新座席状态 |

座席面向前端的 HTTP 接口挂在 `/api/v1/agent`（login、logout、checkin、checkout、
busy、ready、info、status、heartbeat）。

## 4.12 代码结构与配置

### 4.12.1 agentcenter

```
business/services/agentcenter/
├── cmd/main.go                      # 启动、路由、过期会话扫描器
├── configs/config.yaml              # 服务配置
└── internal/
    ├── handler/                     # 座席、分机、技能组、内部管理接口
    ├── service/                     # 状态机编排、会话、Redis 操作
    ├── repository/                  # MySQL：座席、在线记录等
    └── model/                       # 状态枚举、座席、技能组模型
```

座席状态操作不使用 Lua 脚本，状态流转与会话维护由 Go 服务直接完成；座席锁的
比对释放是唯一使用 Lua 的环节。

### 4.12.2 push

```
business/services/push/
├── cmd/main.go                      # 启动、WS 与 HTTP 路由
├── configs/config.yaml
└── internal/
    ├── handler/                     # WS 握手与升级、HTTP 推送接口
    ├── service/                     # 连接管理、路由表、订阅、AgentGateway
    ├── consumer/                    # NATS / Redis 业务事件消费者
    └── model/                       # 事件常量、消息信封、数据结构
```

### 4.12.3 专属配置

| 配置键 | 默认值 | 说明 |
|---|---|---|
| push.agentcenter_base_url | http://127.0.0.1:8081 | 座席命令网关地址 |
| nats.urls | — | 非空时启用 nats 推送模式 |
| 会话 TTL | 120s | 代码默认值 |
| 座席保活周期 | 30s | 代码默认值，连续失败 3 次关闭连接 |
| 过期扫描间隔 | 30s | 代码默认值 |
| WS 读超时 | 60s | pongWait |
