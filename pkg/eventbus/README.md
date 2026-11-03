# eventbus Event Bus

A unified event bus wrapper supporting both NATS and Redis Pub/Sub backends.

## Design Goals

- **Unified interface**: All services use the same event publish/subscribe interface
- **Dual channels**: NATS first, with Redis Pub/Sub fallback
- **Unified format**: All events use the same envelope format
- **Unified naming**: The `vocara.event.{type}` topic naming convention

## Event Format

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

## Event Types

### Agent-Related
- `agent.status_change` — Agent status change
- `agent.checkin` — Agent check-in
- `agent.checkout` — Agent check-out

### Call-Related
- `call.ringing` — Incoming call ringing
- `call.answered` — Call answered
- `call.bridged` — Call bridged
- `call.held` — Call on hold
- `call.unheld` — Call resumed
- `call.hangup` — Call ended
- `call.transferred` — Call transferred

### Queueing-Related
- `queue.enqueue` — Call enqueued
- `queue.dequeue` — Call dequeued (assigned successfully)
- `queue.abandoned` — Queueing abandoned
- `queue.timeout` — Queueing timed out
- `queue.position` — Queue position updated

### CDR-Related
- `cdr.created` — CDR created

### System-Related
- `system.notice` — System notification

## Usage Examples

### Publishing Events

```go
import "vocara/pkg/eventbus"

// Create a publisher (NATS first, with Redis fallback)
publisher, _ := eventbus.NewDualPublisher(natsPub, redisPub)

// Publish an event
event := eventbus.NewEvent(
    eventbus.EventAgentStatusChange,
    tenantID,
    "agentcenter",
    statusData,
)
publisher.Publish(ctx, event)
```

### Subscribing to Events

```go
// NATS subscription
subscriber := eventbus.NewNATSSubscriber(natsConn)
subscriber.Subscribe(ctx, eventbus.EventAgentStatusChange, handler)

// Subscribe to all events
subscriber.SubscribeAll(ctx, handler)
```

## Topic Naming

Full topic = `vocara.event.` + event type

For example:
- `vocara.event.agent.status_change`
- `vocara.event.call.hangup`
- `vocara.event.queue.enqueue`
