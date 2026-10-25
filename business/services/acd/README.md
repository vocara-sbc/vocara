# Vocara ACD Service

Vocara 呼叫中心 **ACD（Automatic Call Distribution）** 排队服务，负责将来电智能分配给空闲座席。

## 功能特性

- ✅ **双 ZSET 队列**：活跃队列 + 暂停队列，支持 IVR 中转、分配待确认等场景
- ✅ **事件驱动调度**：入队/座席就绪/恢复等事件触发即时分配，毫秒级响应
- ✅ **优先级 + 老化机制**：VIP 优先，等待越久优先级越高，防饥饿
- ✅ **分布式锁**：同队列同一时刻只有一个节点在分配，避免重复分配
- ✅ **Lua 原子操作**：所有 Redis 操作均通过 Lua 脚本保证原子性
- ✅ **实时监控**：队列长度、等待时长、SLA 等核心指标实时查询
- ✅ **定时兜底**：每 5 秒扫描一次，确保事件丢失时仍能正常分配

## 技术架构

### 核心方案

采用 **Redis 双 ZSET + 事件驱动调度** 方案（方案四）：

```
┌───────────────────────────────────────────────────────────┐
│                        ACD 服务                            │
│                                                           │
│  ┌──────────┐   Enqueue   ┌──────────┐   Allocate        │
│  │  Call    │ ──────────▶ │  Active  │ ────────▶ 座席     │
│  │  Ctrl    │             │  Queue   │                    │
│  └──────────┘             └────┬─────┘                    │
│                                │ Pause/Resume             │
│                                ▼                           │
│                           ┌──────────┐                    │
│                           │  Paused  │                    │
│                           │  Queue   │ (IVR中/待确认等)    │
│                           └──────────┘                    │
└───────────────────────────────────────────────────────────┘
```

### Redis 数据结构

| Key | 类型 | 说明 |
|-----|------|------|
| `vocara:acd:queue:active:{queueId}` | ZSET | 活跃排队队列，score = priority * 10^12 + eqTimestamp |
| `vocara:acd:queue:paused:{queueId}` | ZSET | 暂停分配队列，score = pauseTimestamp |
| `vocara:acd:queue:call:{callId}` | HASH | 通话详情 |
| `vocara:acd:queue:working-ids` | SET | 工作中队列 ID 集合 |
| `vocara:acd:queue:stats:{queueId}` | HASH | 队列统计数据 |
| `vocara:acd:queue:lock:{queueId}` | STRING | 队列分配分布式锁 |

### 优先级算法

```
score = priority * 10^12 + eqTimestamp
```

- 优先级分为 3 级：VIP(3) > 普通(2) > 低优(1)
- 同优先级下，入队时间越早（timestamp 越小），排名越靠前
- 每次分配时检查老化：每等待 `priorityAgingCycle` 秒，优先级提升一级
- 优先级最高不超过 `maxPriority`

## 项目结构

```
services/acd/
├── go.mod                    # Go module: vocara/service/acd
├── cmd/
│   └── main.go               # 服务启动入口
├── internal/
│   ├── handler/              # HTTP Handler (Hertz)
│   │   ├── acd_handler.go    # ACD 接口处理器
│   │   └── monitor_handler.go # 监控接口处理器
│   ├── service/              # 业务逻辑层
│   │   ├── acd_service.go    # ACD 核心服务
│   │   ├── scheduler.go      # 事件驱动调度器
│   │   └── monitor_service.go # 监控服务
│   ├── repository/           # 数据访问层
│   │   ├── acd_redis_repo.go # Redis 操作（Lua 脚本）
│   │   └── stats_repo.go     # 统计数据仓储
│   └── model/                # 数据模型
│       ├── call.go           # 排队通话模型
│       ├── queue.go          # 队列模型
│       └── stats.go          # 统计模型
├── scripts/                  # Lua 脚本（原子操作）
│   ├── enqueue.lua           # 入队
│   ├── allocate.lua          # 分配（取队首）
│   ├── pause.lua             # 暂停
│   ├── resume.lua            # 恢复
│   └── dequeue.lua           # 出队
├── configs/
│   └── config.yaml           # 配置文件
└── README.md
```

## API 接口

### 基础信息

- Base URL: `http://<host>:<port>/api/v1/acd`
- Content-Type: `application/json`
- Trace ID: 通过 `X-Trace-Id` 请求头透传

### 1. 通话入队

**POST** `/enqueue`

请求体：
```json
{
  "callId": "call-001",
  "caller": "13800138000",
  "called": "10086",
  "queueId": 1,
  "skillGroupId": 101,
  "priority": 2,
  "tenantId": 1,
  "customerInfo": "{\"name\":\"张三\",\"level\":\"gold\"}",
  "uui": "随路数据"
}
```

响应：
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

### 2. 通话出队

**POST** `/dequeue`

请求体：
```json
{
  "callId": "call-001",
  "reason": "answered"   // answered | abandoned | transferred | timeout | error
}
```

### 3. 暂停分配

**POST** `/pause`

请求体：
```json
{
  "callId": "call-001",
  "reason": "ivr"
}
```

### 4. 恢复分配

**POST** `/resume`

请求体：
```json
{
  "callId": "call-001"
}
```

### 5. 查询通话信息

**GET** `/call/{callId}`

响应：
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

### 6. 单队列监控

**GET** `/monitor/queue/{queueId}`

### 7. 全部队列监控

**GET** `/monitor/queues?tenantId=1`

响应：
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

### 8. 健康检查

**GET** `/health`

## 配置说明

完整配置见 `configs/config.yaml`（本机标准配置）。开发联调环境使用 `configs/config.local.yaml`（指向共享联调主机，不进版本库），通过 `CONFIG_PATH=configs/config.local.yaml` 加载。

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `acd.maxQueueLen` | int | 100 | 最大排队长度，0 为不限制 |
| `acd.maxPriority` | int | 3 | 最高优先级 |
| `acd.priorityAgingCycle` | int | 60 | 优先级老化周期（秒），0 关闭 |
| `acd.maxWaitTime` | int | 600 | 最大等待时间（秒），超时出队 |
| `acd.scheduleInterval` | int | 5 | 定时兜底扫描间隔（秒） |
| `acd.lockTimeout` | int | 10 | 分布式锁超时（秒） |
| `acd.allocateMaxRetries` | int | 3 | 分配最大重试次数 |

## 事件驱动

### 订阅的事件

| 事件主题 | 发布者 | 触发动作 |
|----------|--------|----------|
| `vocara.acd.call.enqueue` | ACD 服务 | 触发对应队列分配 |
| `vocara.acd.call.resume` | ACD 服务 | 触发对应队列分配 |
| `vocara.agent.ready` | AgentCenter | 触发对应队列分配 |

### 发布的事件

| 事件主题 | 触发时机 |
|----------|----------|
| `vocara.acd.call.enqueue` | 通话入队成功 |
| `vocara.acd.call.dequeue` | 通话出队（各种原因） |
| `vocara.acd.call.resume` | 通话恢复分配 |

事件消息格式（JSON）：
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

## 本地开发

### 环境要求

- Go 1.22+
- Redis 6.0+

### 启动服务

```bash
cd services/acd

# 使用 go.work 模式（推荐）
cd ../..
go work use ./services/acd

# 运行
cd services/acd
go run ./cmd
```

### 测试

```bash
# 入队测试
curl -X POST http://localhost:8081/api/v1/acd/enqueue \
  -H "Content-Type: application/json" \
  -d '{"callId":"test-001","caller":"13800138000","called":"10086","queueId":1,"skillGroupId":101,"priority":2,"tenantId":1}'

# 查询队列状态
curl http://localhost:8081/api/v1/acd/monitor/queue/1

# 健康检查
curl http://localhost:8081/health
```

## 设计要点

### 1. 为什么用双 ZSET？

- **Active 队列**：正常等待分配的通话，按优先级+等待时间排序
- **Paused 队列**：暂停分配的通话（IVR 中、已分配待确认等），按暂停时间排序
- 分离两个队列避免暂停通话占用 active 队首，影响分配效率

### 2. 为什么用事件驱动 + 定时兜底？

- **事件驱动**：入队/座席就绪时立即触发分配，延迟低
- **定时兜底**：防止事件丢失或异常场景下队列堆积
- 两者结合：既保证低延迟，又保证可靠性

### 3. 为什么用分布式锁？

- ACD 服务多节点部署时，同一队列可能被多个节点同时分配
- 队列级分布式锁保证同一时刻只有一个节点在分配该队列
- 避免重复分配（同一个通话分配给多个座席）

### 4. 老化机制的实现

- 不在后台定时任务中批量更新 score（性能差）
- 而是在每次 allocate 时检查队首通话是否需要提升优先级
- 需要提升时才更新 score 并重新排序，返回 `PRIORITY_UPDATED_RETRY`
- 调用方重试一次即可获取更新后的队首

## 依赖公共库

- `vocara/pkg/config` - 配置管理
- `vocara/pkg/logger` - 日志（支持 trace_id 透传）
- `vocara/pkg/errors` - 统一错误类型
- `vocara/pkg/redis` - Redis 客户端封装
- `vocara/pkg/response` - 统一 HTTP 响应格式
- `vocara/pkg/database` - 数据库访问
- `vocara/pkg/utils` - 通用工具函数

## License

MIT
