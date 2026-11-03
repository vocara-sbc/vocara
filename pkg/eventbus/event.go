// Package eventbus provides a unified event bus wrapper.
// It supports NATS JetStream and Redis Pub/Sub backends, with NATS preferred and Redis as fallback.
package eventbus

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ===== Event type constants =====
// Type definitions for all business events, globally unique

const (
	// ===== Agent events =====
	EventAgentStatusChange = "agent.status_change" // Agent status change
	EventAgentCheckIn      = "agent.checkin"       // Agent check-in
	EventAgentCheckOut     = "agent.checkout"      // Agent check-out

	// ===== Call events =====
	EventCallRinging    = "call.ringing"    // Incoming call ringing
	EventCallAnswered   = "call.answered"   // Call answered
	EventCallBridged    = "call.bridged"    // Call bridged
	EventCallHeld       = "call.held"       // Call held
	EventCallUnheld     = "call.unheld"     // Call resumed
	EventCallHangup     = "call.hangup"     // Call ended
	EventCallTransferred = "call.transferred" // Call transferred

	// ===== Queueing events =====
	EventQueueEnqueue    = "queue.enqueue"    // Call enqueued
	EventQueueDequeue    = "queue.dequeue"    // Call dequeued after successful allocation
	EventQueueAbandoned  = "queue.abandoned"  // Queueing abandoned
	EventQueueTimeout    = "queue.timeout"    // Queueing timeout
	EventQueuePosition   = "queue.position"   // Queue position updated

	// ===== CDR events =====
	EventCDRCreated = "cdr.created" // CDR created

	// ===== System events =====
	EventSystemNotice = "system.notice" // System notice

	// ===== Inbound route events =====
	// admin publishes this after creating, deleting, modifying, enabling, or disabling an inbound route; callctrl subscribes and refreshes its route cache immediately
	EventInboundRouteChange = "inbound_route.change" // Inbound route change
)

// ===== Topic naming =====
// NATS and Redis use dot-separated topic names
// Full topic = prefix + event_type
// Example: vocara.event.agent.status_change

const (
	// TopicPrefix is the event topic prefix.
	TopicPrefix = "vocara.event."
)

// GetTopic returns the full topic for an event type.
func GetTopic(eventType string) string {
	return TopicPrefix + eventType
}

// ===== Event envelope format =====

// Event is the event envelope that wraps all business events.
type Event struct {
	EventID   string      `json:"event_id"`   // Unique event ID
	EventType string      `json:"event_type"` // Event type
	Timestamp int64       `json:"timestamp"`  // Event timestamp in milliseconds
	TenantID  uint64      `json:"tenant_id"`  // Tenant ID
	Source    string      `json:"source"`     // Source service name
	Data      interface{} `json:"data"`       // Event data with a structure specific to the event type
}

// NewEvent creates an event.
func NewEvent(eventType string, tenantID uint64, source string, data interface{}) *Event {
	return &Event{
		EventID:   "evt_" + uuid.New().String()[:16],
		EventType: eventType,
		Timestamp: time.Now().UnixMilli(),
		TenantID:  tenantID,
		Source:    source,
		Data:      data,
	}
}

// Topic returns the event's topic.
func (e *Event) Topic() string {
	return GetTopic(e.EventType)
}

// ToJSON serializes the event to JSON.
func (e *Event) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// ParseEvent parses an event from JSON.
func ParseEvent(data []byte) (*Event, error) {
	var evt Event
	if err := json.Unmarshal(data, &evt); err != nil {
		return nil, fmt.Errorf("parse event failed: %w", err)
	}
	return &evt, nil
}

// ===== Publisher interface =====

// Publisher is the event publisher interface.
type Publisher interface {
	// Publish publishes an event.
	Publish(ctx context.Context, event *Event) error
	// Close closes the connection.
	Close() error
}

// ===== Subscriber interface =====

// Handler handles an event.
type Handler func(ctx context.Context, event *Event) error

// Subscriber is the event subscriber interface.
type Subscriber interface {
	// Subscribe subscribes to the specified event type.
	Subscribe(ctx context.Context, eventType string, handler Handler) (string, error)
	// Unsubscribe cancels a subscription.
	Unsubscribe(ctx context.Context, subscriptionID string) error
	// Close closes the connection.
	Close() error
}

// ===== Event bus interface =====

// EventBus is the event bus interface for publishing and subscribing.
type EventBus interface {
	Publisher
	Subscriber
}
