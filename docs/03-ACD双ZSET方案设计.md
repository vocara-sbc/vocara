# 03-ACD 双 ZSET 方案设计

> Vocara ACD（Automatic Call Distribution，自动呼叫分配）负责把呼入通话按
> 队列组织起来，在通话与座席之间完成匹配。本文档描述 ACD 的 Redis 数据结构、
> Lua 原子脚本、事件驱动调度器、分布式锁与统计模型。
> 代码位于 `business/services/acd`（module `vocara/services/acd`）。

## 3.1 排队模型

### 3.1.1 双 ZSET 结构

每个队列在 Redis 中持有两个有序集合：

| ZSET | 含义 | 成员 | score |
|---|---|---|---|
| active | 等待分配的通话，参与正常排队 | callId | 优先级与入队时间的组合值 |
| paused | 暂停分配的通话（通话进入 IVR 等暂不分配的场景） | callId | 暂停时刻时间戳 |

active 中 score 最小的成员位于队首，即当前应被优先分配的通话。
通话从 active 中窥视取出，**只有分配成功或通话终结才真正移除**；
paused 中的通话不参与分配，恢复时按入队时间重新计算 score 回到 active，
保持其原始排队位置。

### 3.1.2 辅助结构

双 ZSET 之外，每个队列还配有通话详情 HASH、统计 HASH 与队列级分布式锁；
全局有一个工作队列集合（working-ids）记录存在等待通话的队列，供定时扫描使用。

### 3.1.3 设计目标

| 目标 | 实现方式 |
|---|---|
| 分配操作原子性 | 入队、窥视、暂停、恢复、出队均由 Lua 脚本一次执行完成 |
| 高优先级与公平等待兼顾 | score 高位编码优先级、低位编码入队时间，同优先级按等待时长排序 |
| 长时间等待自动提权 | 优先级老化：在 allocate 脚本内按等待时长提升当前优先级 |
| 事件实时触发 | 入队、恢复、座席就绪经 Redis Pub/Sub 即时触发分配 |
| 触发丢失可兜底 | 定时扫描 working-ids 集合，周期性补偿触发分配 |
| 多节点安全 | 队列级分布式锁保证同一时刻只有一个节点执行分配 |
| 队列长度可控 | `max_queue_len` 限制单队列等待容量，默认 100 |

## 3.2 Redis 数据结构

### 3.2.1 Key 总览

| Key | 类型 | 粒度 | 说明 |
|---|---|---|---|
| `vocara:acd:queue:active:{queueId}` | ZSET | 队列 | 等待分配的通话 |
| `vocara:acd:queue:paused:{queueId}` | ZSET | 队列 | 暂停分配的通话 |
| `vocara:acd:queue:call:{callId}` | HASH | 通话 | 通话排队详情 |
| `vocara:acd:queue:stats:{queueId}` | HASH | 队列 | 队列当日统计与实时计数 |
| `vocara:acd:queue:working-ids` | SET | 全局 | 存在等待通话的队列 ID 集合 |
| `vocara:acd:queue:lock:{queueId}` | STRING | 队列 | 队列级分布式锁 |

空闲座席集合不属于 ACD：它由座席中心维护
（`vocara:agent:idle:{skillGroupId}`，见 04 文档），ACD 经座席中心 HTTP
接口获取空闲座席，不直接读取该集合。

### 3.2.2 score 编码

```
score = currentPriority × 10^12 + 入队时刻 Unix 秒级时间戳
```

- `currentPriority` 取值 1~3（见 3.7），优先级数值越大越优先；
- 低位时间戳为秒级 Unix 时间，同一优先级内入队越早 score 越小、越靠队首；
- `10^12` 远大于秒级时间戳的取值范围，优先级位与时间位互不进位干扰；
- score 在 Go 层以 `float64` 计算（`acd_redis_repo.go`），在 Lua 内做老化
  重算时按同一公式以整数运算生成。

### 3.2.3 通话详情 HASH 字段

`vocara:acd:queue:call:{callId}` 在入队时写入以下字段：

| 字段 | 说明 |
|---|---|
| `callId` | 通话唯一标识 |
| `caller` | 主叫号码 |
| `called` | 被叫号码 |
| `queueId` | 队列 ID |
| `skillGroupId` | 技能组 ID（向座席中心请求空闲座席的依据） |
| `priority` | 基础优先级（入队时确定） |
| `currentPriority` | 当前优先级（含老化提升，初始等于 `priority`） |
| `status` | 通话在队列中的状态：`waiting` / `paused` / `serving` / `finished` |
| `eqTime` | 入队时刻 Unix 秒级时间戳 |
| `tenantId` | 租户 ID |
| `customerInfo` | 客户信息（JSON 字符串，可为空） |
| `uui` | 随路数据（User-to-User Information，可为空） |

暂停时追加 `pauseTime`、`pauseReason`；恢复时追加 `resumeTime`；
接听出队时改写 `status=serving` 并追加 `dequeueTime`、`dequeueReason`。

### 3.2.4 统计 HASH 字段

`vocara:acd:queue:stats:{queueId}` 保存当日累计指标与计数快照：

| 字段 | 说明 |
|---|---|
| `total_today` | 今日入队总数（脚本外由 `StatsRepo.IncrTotalToday` 累加） |
| `answered_today` | 今日接听数 |
| `abandoned_today` | 今日客户放弃数 |
| `timeout_today` | 今日超时数 |
| `transferred_today` | 今日转接数 |
| `avg_wait_time` | 今日平均等待秒数（接听时增量计算，保留两位小数） |
| `sla_answered` | 今日 20 秒内接听数 |
| `sla` | 今日服务水平百分比（20 秒内接听占比，保留两位小数） |
| `serving` | 通话中计数 |
| `waiting` / `paused` | 等待 / 暂停计数快照（出队时刷新；实时数以 ZCARD 为准） |

`waiting`、`paused` 的快照值仅在出队脚本中刷新，查询实时统计时
`StatsRepo.GetQueueStats` 以 active、paused 的 `ZCARD` 结果覆盖。

### 3.2.5 Key 过期策略

| Key | 过期策略 |
|---|---|
| active / paused ZSET | 不设 TTL，成员由出队脚本移除 |
| 通话详情 HASH（waiting/paused） | 不设 TTL，随出队删除 |
| 通话详情 HASH（serving） | 接听出队后保留 3600 秒，供通话中查询，到期自动删除 |
| 统计 HASH | 不设 TLL，每日由 `ResetDailyStats` 清零日累计字段 |
| working-ids | 不设 TTL，由出队脚本在两队列均空时移除队列 ID |
| 锁 | 写入时带 10 秒 TTL（见 3.5） |

## 3.3 Lua 原子脚本

五个脚本通过 `business/services/acd/scripts` 包以 `//go:embed` 编译期嵌入，
由 `AcdRedisRepo` 预编译为 `redis.Script` 调用。

### 3.3.1 enqueue.lua（入队）

| 项 | 内容 |
|---|---|
| KEYS | active ZSET、通话 HASH、working-ids SET |
| ARGV | score、callId、caller、called、queueId、skillGroupId、priority、eqTime、tenantId、customerInfo、uui、maxQueueLen |

执行逻辑：

1. `EXISTS` 通话 HASH，已存在返回 `CALL_ALREADY_EXISTS`，防止同一通话重复入队；
2. `maxQueueLen > 0` 时 `ZCARD` active，达到容量上限返回 `QUEUE_FULL`；
3. `ZADD` active（score → callId）；
4. `HSET` 通话详情 12 个字段，`status=waiting`、`currentPriority=priority`；
5. `SADD` working-ids 标记队列工作中；
6. 返回 `ENQUEUED` 与 score。

入队成功后，`total_today` 由 Go 层调用 `StatsRepo.IncrTotalToday` 累加，
不在入队脚本内完成。

### 3.3.2 allocate.lua（窥视队首 + 老化）

| 项 | 内容 |
|---|---|
| KEYS | active ZSET、paused ZSET、通话 HASH 前缀（`vocara:acd:queue:call:`） |
| ARGV | 当前时间戳、老化周期（秒）、最大优先级 |

执行逻辑（循环）：

1. `ZRANGE active 0 0 WITHSCORES` 窥视队首，空队列返回 `EMPTY_QUEUE`；
2. 通话 HASH 不存在视为脏数据，`ZREM` 后继续窥视下一个；
3. `HMGET` 通话详情，`status` 不为 `waiting` 的成员 `ZREM` 后继续；
4. 优先级老化处理（见 3.7.4）：当前优先级应提升时，重算 score 并
   `ZADD` 更新排序、`HSET currentPriority`，返回 `PRIORITY_UPDATED_RETRY`，
   由调用方重新窥视；
5. 无需老化时返回 `ok` 及通话详情（callId、score、主被叫、队列与技能组、
   基础与当前优先级、入队时间、租户、客户信息、随路数据、已等待秒数）。

allocate 是 **peek 语义**：只读取不移除，通话在分配成功后由 dequeue 移除。
paused ZSET 仅作为 KEYS 传入以保证同队列脚本路由到同一 Redis 节点，
脚本本身不读取它。

### 3.3.3 pause.lua（暂停）

| 项 | 内容 |
|---|---|
| KEYS | active ZSET、paused ZSET、通话 HASH |
| ARGV | callId、暂停时间戳、暂停原因（可空） |

执行逻辑：

1. 通话 HASH 不存在返回 `CALL_NOT_FOUND`；
2. 状态为 `paused` 返回 `ALREADY_PAUSED`，不为 `waiting` 返回 `INVALID_STATUS`；
3. `ZREM` active，移除失败返回 `NOT_IN_ACTIVE_QUEUE`；
4. `ZADD` paused，score 为暂停时刻时间戳；
5. `HSET status=paused、pauseTime`，原因非空时写入 `pauseReason`。

### 3.3.4 resume.lua（恢复）

| 项 | 内容 |
|---|---|
| KEYS | active ZSET、paused ZSET、通话 HASH、working-ids SET |
| ARGV | callId、恢复时间戳 |

执行逻辑：

1. 通话 HASH 不存在返回 `CALL_NOT_FOUND`；
2. 状态为 `waiting` 返回 `ALREADY_ACTIVE`，不为 `paused` 返回 `INVALID_STATUS`；
3. `ZREM` paused，移除失败返回 `NOT_IN_PAUSED_QUEUE`；
4. 读取 `eqTime` 与 `currentPriority`，按
   `currentPriority × 10^12 + eqTime` 重算 score，`ZADD` active，
   保持通话以入队时刻为准的原始排队位置；
5. `HSET status=waiting、resumeTime`，`SADD` working-ids；
6. 返回 `RESUMED` 与新 score。

### 3.3.5 dequeue.lua（出队）

| 项 | 内容 |
|---|---|
| KEYS | active ZSET、paused ZSET、通话 HASH、统计 HASH、working-ids SET |
| ARGV | callId、出队原因、出队时间戳 |

出队原因取值：

| reason | 语义 |
|---|---|
| `answered` | 座席接听 |
| `abandoned` | 客户放弃（挂机） |
| `transferred` | 转接出队 |
| `timeout` | 等待超时 |
| `error` | 异常出队 |

执行逻辑：

1. 通话 HASH 不存在返回 `CALL_NOT_FOUND`；
2. 按 HASH 中的 `status` 从 active（waiting）或 paused（paused）`ZREM`；
3. 等待秒数 = 出队时间 − `eqTime`（负值归零）；
4. 按 reason 更新统计：
   - `answered`：`answered_today +1`、增量计算 `avg_wait_time`、
     `serving +1`；等待 ≤ 20 秒时 `sla_answered +1`，并重算 `sla` 百分比；
   - `abandoned` / `timeout` / `transferred`：对应计数 +1；
5. 通话 HASH 处理：`answered` 改写 `status=serving` 并设置 3600 秒过期；
   其他原因直接 `DEL`；
6. active 与 paused 均空时，从 working-ids `SREM` 该队列；
7. 把 active、paused 的 `ZCARD` 结果写入统计 HASH 的 `waiting`、`paused`；
8. 返回 `DEQUEUED` 与等待秒数。

## 3.4 事件驱动调度器

调度器 `Scheduler`（`internal/service/scheduler.go`）在启动时运行三个
协程，构成"事件触发 + 触发合并 + 定时兜底"的调度结构。

### 3.4.1 内部事件频道（Redis Pub/Sub）

`subscribeEvents` 协程订阅三个 Redis 频道：

| 频道 | 发布者 | 载荷要点 | 动作 |
|---|---|---|---|
| `vocara.acd.call.enqueue` | ACD 入队流程 | callId、queueId、技能组、优先级、租户、时间戳 | 触发该队列分配 |
| `vocara.acd.call.resume` | ACD 恢复流程 | callId、queueId、租户、时间戳 | 触发该队列分配 |
| `vocara.agent.ready` | 座席中心 | agentId、skillGroupId、tenantId，可携带 queueId | 携带 queueId 时触发该队列 |

内部频道仅用于 ACD 进程内的实时触发，采用 Redis Pub/Sub。
面向其他服务的业务事件走统一事件总线（见 3.4.5）。

### 3.4.2 触发合并

- 事件处理只把队列 ID 写入 `triggerCh`（缓冲 1000），通道满时丢弃，
  丢弃的触发由定时扫描兜底；
- `processTriggers` 协程以 100 毫秒为周期取出通道内的队列 ID，
  在 map 中按队列去重合并，每个被触发的队列启动一个 goroutine 执行分配。

### 3.4.3 定时兜底扫描

`scheduleLoop` 协程按 `schedule_interval`（默认 5 秒）周期读取
working-ids 集合，把所有工作中队列 ID 重新写入 `triggerCh`。
该机制保证任何丢失的事件触发都能在下一个扫描周期得到补偿。

### 3.4.4 分配循环

`allocateQueue` 执行单个队列的一轮分配：

1. `TryLock` 获取队列级分布式锁，未获取说明其他节点正在分配，直接返回；
2. 进入分配循环（单轮上限 100 次，防止持锁时间超过锁 TTL）：
   1. 调用 allocate.lua 窥视队首；收到 `PRIORITY_UPDATED_RETRY` 时重新窥视；
      脚本报错时终止本轮；队列为空时结束；
   2. 以通话的技能组 ID、租户 ID 调用座席中心 `GetReadyAgent` 请求空闲座席；
      座席中心异常或无空闲座席时结束本轮——allocate 为 peek 语义，
      通话保留在 active 中等待下次触发；
   3. 调用座席中心 `AssignCall` 把通话指派给座席；指派失败时调用
      `ResumeReady` 恢复该座席为就绪，继续尝试下一个空闲座席；
   4. 指派成功后调用 dequeue.lua 完成出队；
   5. 发布分配成功事件（见 3.4.5）。

### 3.4.5 统一事件总线事件

| 事件类型 | 发布时机 | 订阅方用途 |
|---|---|---|
| `queue.enqueue` | 入队成功 | 推送队列位置变化 |
| `queue.dequeue` | 分配成功、接听出队 | callctrl 对座席分机发起振铃并桥接客户腿；推送状态 |
| `queue.abandoned` | 客户放弃出队 | 推送与统计 |
| `queue.timeout` | 超时出队 | 推送与统计 |

分配成功事件（`queue.dequeue`）的载荷为契约 `event.AllocatedCall`：
callId、queueId、skillGroupId、tenantId、agentId、dnNumber、caller、called。
该事件以异步 goroutine 发布，3 秒超时，发布失败仅记录日志。

## 3.5 队列级分布式锁

| 项 | 内容 |
|---|---|
| Key | `vocara:acd:queue:lock:{queueId}` |
| 获取 | `SET lockKey "1" NX EX 10`（`lock_timeout`，默认 10 秒） |
| 释放 | 分配循环结束后 `DEL` 锁 Key |
| 持锁上限 | 单轮分配循环最多 100 次，正常执行时长远小于锁 TTL |

锁不做持有者校验：获取锁的节点在分配循环结束（含异常退出路径上的
defer）时直接删除锁；节点崩溃导致未释放时，锁在 TTL 到期后自动释放，
其他节点可继续分配。锁只保护"同一队列同一时刻仅一个节点执行分配循环"，
座席超卖由座席中心的状态机与二次校验保证（见 04、05 文档）。

## 3.6 统计与监控

### 3.6.1 实时统计查询

`StatsRepo.GetQueueStats` 汇总队列实时统计：

- 统计 HASH 中的当日累计字段（total/answered/abandoned、avg_wait_time、
  serving、sla）；
- active、paused 的实时 `ZCARD`；
- active 非空时遍历成员通话 HASH 的 `eqTime`，计算当前最长等待时间
  `MaxWaitTime` 与对应通话 `LongestCallID`；
- 返回 `model.QueueStats`。

### 3.6.2 监控接口

队列监控不经过指标拉取组件，由 `MonitorService`（`internal/service/monitor_service.go`）
经 HTTP 接口提供：队列列表、单队列实时统计、通话详情查询，供管理后台
"系统监控 / 实时监控"页面与监控大屏直接查询。

### 3.6.3 每日清零

`ResetDailyStats` 在清零周期内把日累计字段（total_today、answered_today、
abandoned_today、timeout_today、transferred_today、avg_wait_time、sla、
sla_answered）重置为 0，serving 与实时队列数不受影响。

## 3.7 优先级模型

### 3.7.1 优先级枚举

| 枚举 | 值 | 说明 |
|---|---|---|
| `PriorityLow` | 1 | 低优先级 |
| `PriorityNormal` | 2 | 普通优先级（默认） |
| `PriorityVIP` | 3 | VIP 优先级 |

### 3.7.2 入队规范化

入队请求未显式给出优先级（≤ 0）时取 `PriorityNormal`；
优先级大于 `max_priority`（默认 3）时截断为 `max_priority`。

### 3.7.3 score 与排序

优先级占据 score 的高位，因此任何高优先级通话都排在低优先级之前；
同一优先级内按入队秒级时间戳升序，先入队者先分配。

### 3.7.4 优先级老化

老化在 allocate.lua 窥视队首时内联计算：

```
提升后优先级 = min(基础优先级 + floor(已等待秒数 / priority_aging_cycle), max_priority)
```

- `priority_aging_cycle` 默认 60 秒，设为 0 关闭老化；
- 仅当计算结果大于 `currentPriority` 且小于 `max_priority` 时处理；
- 提升时重算 score 并更新 active 排序与 HASH 的 `currentPriority`，
  随后返回 `PRIORITY_UPDATED_RETRY` 重新窥视队首；
- 老化只改变 `currentPriority`，基础优先级 `priority` 保持不变；
- 等待时间以秒计，按老化周期逐级提升直至 `max_priority` 封顶。

## 3.8 代码结构

```
business/services/acd/
├── cmd/main.go                      # 启动入口：装配 Redis、事件总线、座席客户端、服务与调度器
├── configs/config.yaml              # acd 配置节（容量/优先级/老化/锁/扫描间隔等）
├── scripts/                         # Lua 脚本嵌入包（//go:embed）
│   ├── scripts.go
│   ├── enqueue.lua
│   ├── allocate.lua
│   ├── pause.lua
│   ├── resume.lua
│   └── dequeue.lua
└── internal/
    ├── handler/
    │   ├── acd_handler.go           # ACD 操作 HTTP 接口
    │   └── monitor_handler.go       # 队列监控 HTTP 接口
    ├── service/
    │   ├── acd_service.go           # 入队/出队/暂停/恢复业务逻辑、事件发布
    │   ├── scheduler.go             # 事件订阅、触发合并、定时兜底、分配循环
    │   ├── monitor_service.go       # 监控查询业务逻辑
    │   └── agent_client.go          # 座席中心 HTTP 客户端（GetReadyAgent/AssignCall/ResumeReady）
    ├── repository/
    │   ├── acd_redis_repo.go        # Key 构造、Lua 脚本调用、锁、Pub/Sub
    │   └── stats_repo.go            # 统计 HASH 查询、日累计、清零
    └── model/
        ├── call.go                  # CallInfo、Priority、CallStatus
        ├── queue.go                 # 队列模型
        └── stats.go                 # QueueStats
```

### 3.8.1 配置项

`config.yaml` 的 `acd` 配置节：

| 配置键 | 默认值 | 说明 |
|---|---|---|
| `max_queue_len` | 100 | 单队列等待上限，0 不限制 |
| `max_priority` | 3 | 优先级上限 |
| `priority_aging_cycle` | 60 | 老化周期（秒），0 关闭老化 |
| `max_wait_time` | 600 | 最大等待时间（秒）配置项 |
| `schedule_interval` | 5 | 定时兜底扫描间隔（秒） |
| `lock_timeout` | 10 | 分布式锁 TTL（秒） |
| `allocate_max_retries` | 3 | 分配重试配置项 |
| `agent_center_addr` | http://127.0.0.1:8081 | 座席中心地址 |
| `agent_center_token` | — | 座席中心内部鉴权 Token |
| `use_mock_agent_client` | false | 使用模拟座席客户端（开发测试） |

## 3.9 容量与复杂度

| 操作 | Redis 时间复杂度 |
|---|---|
| 入队（ZADD + HSET + SADD） | O(log N) |
| 窥视队首（ZRANGE 0 0） | O(log N) |
| 出队（ZREM + HSET） | O(log N) |
| 队列长度（ZCARD） | O(1) |
| 单队列实时统计 | ZSET/HASH 操作为 O(1)~O(log N)；最长等待遍历为 O(N) |

N 为单队列等待通话数。等待容量受 `max_queue_len` 约束（默认 100），
active、paused、working-ids、stats 等结构的空间占用均与队列数和
队列内通话数成线性关系。
