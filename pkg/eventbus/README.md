# eventbus 事件总线

统一的事件总线封装，支持 NATS 和 Redis Pub/Sub 两种后端。

## 设计目标

- **统一接口**：所有服务使用相同的事件发布/订阅接口
- **双通道**：NATS 优先，Redis Pub/Sub 降级
- **统一格式**：所有事件使用相同的信封格式
- **统一命名**：`vocara.event.{type}` 主题命名规范

## 事件格式

```json
{
  "event_id": "evt_xxx",
  "event_type": "agent.status_change",
  "timestamp": 1704067200000,
  "tenant_id": 1,
  "source": "agentcenter",
  "data": { ... }
}
```

## 事件类型

### 座席相关
- `agent.status_change` — 座席状态变更
- `agent.checkin` — 座席签入
- `agent.checkout` — 座席签出

### 通话相关
- `call.ringing` — 来电振铃
- `call.answered` — 通话接通
- `call.bridged` — 通话桥接
- `call.held` — 通话保持
- `call.unheld` — 通话恢复
- `call.hangup` — 通话结束
- `call.transferred` — 通话转接

### 排队相关
- `queue.enqueue` — 通话入队
- `queue.dequeue` — 通话出队（分配成功）
- `queue.abandoned` — 排队放弃
- `queue.timeout` — 排队超时
- `queue.position` — 排队位置更新

### CDR 相关
- `cdr.created` — CDR 生成

### 系统相关
- `system.notice` — 系统通知

## 使用示例

### 发布事件

```go
import "vocara/pkg/eventbus"

// 创建发布者（NATS 优先，Redis 降级）
publisher, _ := eventbus.NewDualPublisher(natsPub, redisPub)

// 发布事件
event := eventbus.NewEvent(
    eventbus.EventAgentStatusChange,
    tenantID,
    "agentcenter",
    statusData,
)
publisher.Publish(ctx, event)
```

### 订阅事件

```go
// NATS 订阅
subscriber := eventbus.NewNATSSubscriber(natsConn)
subscriber.Subscribe(ctx, eventbus.EventAgentStatusChange, handler)

// 订阅所有事件
subscriber.SubscribeAll(ctx, handler)
```

## 主题命名

完整主题 = `vocara.event.` + 事件类型

例如：
- `vocara.event.agent.status_change`
- `vocara.event.call.hangup`
- `vocara.event.queue.enqueue`
