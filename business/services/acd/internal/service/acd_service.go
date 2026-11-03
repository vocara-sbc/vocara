// Package service provides the core ACD business logic.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"vocara/pkg/errors"
	"vocara/pkg/eventbus"
	"vocara/pkg/logger"
	"vocara/services/acd/internal/model"
	"vocara/services/acd/internal/repository"
)

// AcdService defines the ACD service interface.
type AcdService interface {
	// Enqueue adds a call to a queue.
	Enqueue(ctx context.Context, req *EnqueueRequest) (*model.CallInfo, error)
	// Dequeue removes a call from a queue, such as after hang up, transfer, or successful allocation.
	Dequeue(ctx context.Context, callID string, reason string) error
	// Pause pauses call allocation, such as when the call enters an IVR.
	Pause(ctx context.Context, callID string, reason string) error
	// Resume resumes call allocation.
	Resume(ctx context.Context, callID string) error
	// GetCallInfo returns call information.
	GetCallInfo(ctx context.Context, callID string) (*model.CallInfo, error)
}

// EnqueueRequest contains an enqueue request.
type EnqueueRequest struct {
	CallID       string         `json:"callId"`
	Caller       string         `json:"caller"`
	Called       string         `json:"called"`
	QueueID      int64          `json:"queueId"`
	SkillGroupID int64          `json:"skillGroupId"`
	Priority     model.Priority `json:"priority"`
	TenantID     int64          `json:"tenantId"`
	CustomerInfo string         `json:"customerInfo,omitempty"`
	UUI          string         `json:"uui,omitempty"`
}

// acdService implements the ACD service.
type acdService struct {
	repo        *repository.AcdRedisRepo
	statsRepo   *repository.StatsRepo
	eventBus    EventPublisher   // Event bus for internal scheduling.
	busPublisher eventbus.Publisher // Unified event bus for WebSocket push and other consumers.
	acdConfig   ACDConfig
}

const eventSourceACD = "acd"

// ACDConfig contains ACD configuration.
type ACDConfig struct {
	// MaxQueueLen is the maximum queue length; 0 means unlimited.
	MaxQueueLen int `json:"maxQueueLen"`
	// MaxPriority is the maximum priority.
	MaxPriority int `json:"maxPriority"`
	// PriorityAgingCycle is the priority aging cycle in seconds.
	PriorityAgingCycle int `json:"priorityAgingCycle"`
	// MaxWaitTime is the maximum wait time in seconds; timeout triggers automatic transfer or queue failure.
	MaxWaitTime int `json:"maxWaitTime"`
}

// EventPublisher defines the event publisher interface.
type EventPublisher interface {
	// Publish publishes an event.
	Publish(ctx context.Context, topic string, event interface{}) error
}

// redisEventPublisher publishes events over Redis Pub/Sub.
type redisEventPublisher struct {
	repo *repository.AcdRedisRepo
}

func (p *redisEventPublisher) Publish(ctx context.Context, topic string, event interface{}) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event failed: %w", err)
	}
	return p.repo.Publish(ctx, topic, string(data))
}

// NewAcdService creates an ACD service.
func NewAcdService(
	repo *repository.AcdRedisRepo,
	statsRepo *repository.StatsRepo,
	eventBus EventPublisher,
	busPublisher eventbus.Publisher,
	config ACDConfig,
) AcdService {
	return &acdService{
		repo:         repo,
		statsRepo:    statsRepo,
		eventBus:     eventBus,
		busPublisher: busPublisher,
		acdConfig:    config,
	}
}

// NewRedisEventPublisher creates a Redis event publisher.
func NewRedisEventPublisher(repo *repository.AcdRedisRepo) EventPublisher {
	return &redisEventPublisher{repo: repo}
}

// Enqueue adds a call to a queue.
func (s *acdService) Enqueue(ctx context.Context, req *EnqueueRequest) (*model.CallInfo, error) {
	// Validate parameters.
	if req.CallID == "" {
		return nil, errors.NewBizError(errors.CodeParamError, "callId is required")
	}
	if req.QueueID <= 0 {
		return nil, errors.NewBizError(errors.CodeParamError, "queueId is required")
	}
	if req.Caller == "" {
		return nil, errors.NewBizError(errors.CodeParamError, "caller is required")
	}
	if req.Priority <= 0 {
		req.Priority = model.PriorityNormal
	}
	if req.Priority > model.Priority(s.acdConfig.MaxPriority) {
		req.Priority = model.Priority(s.acdConfig.MaxPriority)
	}

	// Construct the call information.
	call := &model.CallInfo{
		CallID:          req.CallID,
		Caller:          req.Caller,
		Called:          req.Called,
		QueueID:         req.QueueID,
		SkillGroupID:    req.SkillGroupID,
		Priority:        req.Priority,
		CurrentPriority: req.Priority,
		Status:          model.CallStatusWaiting,
		EQTime:          time.Now(),
		TenantID:        req.TenantID,
		CustomerInfo:    req.CustomerInfo,
		UUI:             req.UUI,
	}

	// Enqueue atomically.
	result, err := s.repo.Enqueue(ctx, call, s.acdConfig.MaxQueueLen)
	if err != nil {
		logger.ErrorCtx(ctx, "enqueue failed",
			zap.String("callId", req.CallID),
			zap.Int64("queueId", req.QueueID),
			zap.Error(err),
		)
		return nil, s.mapLuaError(err)
	}

	logger.InfoCtx(ctx, "call enqueued",
		zap.String("callId", req.CallID),
		zap.Int64("queueId", req.QueueID),
		zap.Int("priority", int(req.Priority)),
		zap.Float64("score", result.Score),
	)

	// Update statistics.
	if err := s.statsRepo.IncrTotalToday(ctx, req.QueueID); err != nil {
		logger.WarnCtx(ctx, "increment total today failed",
			zap.Int64("queueId", req.QueueID),
			zap.Error(err),
		)
	}

	// Publish the enqueue event to trigger immediate allocation by the scheduler.
	s.publishEnqueueEvent(ctx, call)

	return call, nil
}

// validDequeueReasons lists the allowed dequeue reasons, matching the reason values in dequeue.lua.
var validDequeueReasons = map[string]bool{
	"answered": true,
	"abandoned": true,
	"transferred": true,
	"timeout": true,
	"error": true,
}

// Dequeue removes a call from a queue.
func (s *acdService) Dequeue(ctx context.Context, callID string, reason string) error {
	if callID == "" {
		return errors.NewBizError(errors.CodeParamError, "callId is required")
	}
	if !validDequeueReasons[reason] {
		return errors.NewBizError(errors.CodeParamError, "invalid dequeue reason: "+reason)
	}

	// Get the call information first to obtain queueID.
	call, err := s.repo.GetCallInfo(ctx, callID)
	if err != nil {
		logger.ErrorCtx(ctx, "get call info failed before dequeue",
			zap.String("callId", callID),
			zap.Error(err),
		)
		return errors.NewBizError(errors.CodeSystemError, "get call info failed")
	}
	if call == nil {
		return errors.NewBizError(errors.CodeNotFound, "call not found: "+callID)
	}

	// Dequeue atomically.
	result, err := s.repo.Dequeue(ctx, callID, call.QueueID, reason)
	if err != nil {
		logger.ErrorCtx(ctx, "dequeue failed",
			zap.String("callId", callID),
			zap.Int64("queueId", call.QueueID),
			zap.String("reason", reason),
			zap.Error(err),
		)
		return s.mapLuaError(err)
	}

	logger.InfoCtx(ctx, "call dequeued",
		zap.String("callId", callID),
		zap.Int64("queueId", call.QueueID),
		zap.String("reason", reason),
		zap.Int64("waitTime", result.WaitTime),
	)

	// Publish the dequeue event.
	s.publishDequeueEvent(ctx, call, reason, result.WaitTime)

	return nil
}

// Pause pauses call allocation.
func (s *acdService) Pause(ctx context.Context, callID string, reason string) error {
	if callID == "" {
		return errors.NewBizError(errors.CodeParamError, "callId is required")
	}

	// Get the call information.
	call, err := s.repo.GetCallInfo(ctx, callID)
	if err != nil {
		return errors.NewBizError(errors.CodeSystemError, "get call info failed")
	}
	if call == nil {
		return errors.NewBizError(errors.CodeNotFound, "call not found: "+callID)
	}

	// Pause atomically.
	err = s.repo.Pause(ctx, callID, call.QueueID, reason)
	if err != nil {
		logger.ErrorCtx(ctx, "pause failed",
			zap.String("callId", callID),
			zap.Int64("queueId", call.QueueID),
			zap.Error(err),
		)
		return s.mapLuaError(err)
	}

	logger.InfoCtx(ctx, "call paused",
		zap.String("callId", callID),
		zap.Int64("queueId", call.QueueID),
		zap.String("reason", reason),
	)

	return nil
}

// Resume resumes call allocation.
func (s *acdService) Resume(ctx context.Context, callID string) error {
	if callID == "" {
		return errors.NewBizError(errors.CodeParamError, "callId is required")
	}

	// Get the call information.
	call, err := s.repo.GetCallInfo(ctx, callID)
	if err != nil {
		return errors.NewBizError(errors.CodeSystemError, "get call info failed")
	}
	if call == nil {
		return errors.NewBizError(errors.CodeNotFound, "call not found: "+callID)
	}

	// Resume atomically.
	result, err := s.repo.Resume(ctx, callID, call.QueueID)
	if err != nil {
		logger.ErrorCtx(ctx, "resume failed",
			zap.String("callId", callID),
			zap.Int64("queueId", call.QueueID),
			zap.Error(err),
		)
		return s.mapLuaError(err)
	}

	logger.InfoCtx(ctx, "call resumed",
		zap.String("callId", callID),
		zap.Int64("queueId", call.QueueID),
		zap.Float64("score", result.Score),
	)

	// Publish the resume event to trigger allocation by the scheduler.
	s.publishResumeEvent(ctx, call)

	return nil
}

// GetCallInfo returns call information.
func (s *acdService) GetCallInfo(ctx context.Context, callID string) (*model.CallInfo, error) {
	if callID == "" {
		return nil, errors.NewBizError(errors.CodeParamError, "callId is required")
	}

	call, err := s.repo.GetCallInfo(ctx, callID)
	if err != nil {
		return nil, errors.NewBizError(errors.CodeSystemError, "get call info failed")
	}
	if call == nil {
		return nil, errors.NewBizError(errors.CodeNotFound, "call not found: "+callID)
	}

	return call, nil
}

// --- Event publishing ---

// CallEnqueueEvent contains enqueue event data.
type CallEnqueueEvent struct {
	CallID       string         `json:"callId"`
	QueueID      int64          `json:"queueId"`
	SkillGroupID int64          `json:"skillGroupId"`
	Priority     model.Priority `json:"priority"`
	TenantID     int64          `json:"tenantId"`
	Timestamp    int64          `json:"timestamp"`
	TraceID      string         `json:"traceId,omitempty"`
}

// CallResumeEvent contains resume event data.
type CallResumeEvent struct {
	CallID    string `json:"callId"`
	QueueID   int64  `json:"queueId"`
	TenantID  int64  `json:"tenantId"`
	Timestamp int64  `json:"timestamp"`
	TraceID   string `json:"traceId,omitempty"`
}

// CallDequeueEvent contains dequeue event data.
type CallDequeueEvent struct {
	CallID    string `json:"callId"`
	QueueID   int64  `json:"queueId"`
	Reason    string `json:"reason"`
	WaitTime  int64  `json:"waitTime"`
	Timestamp int64  `json:"timestamp"`
	TraceID   string `json:"traceId,omitempty"`
}

// QueueEventData is the unified queue event data structure published to the unified event bus.
// Its fields correspond to CallQueueEventData in the push service.
type QueueEventData struct {
	CallID    string `json:"call_id"`              // Call ID.
	QueueID   int64  `json:"queue_id"`             // Queue ID.
	TenantID  int64  `json:"tenant_id"`            // Tenant ID.
	CallerNum string `json:"caller_num"`           // Calling number.
	CalledNum string `json:"called_num"`           // Called number.
	Priority  int    `json:"priority"`             // Priority.
	Position  int    `json:"position"`             // Queue position.
	Reason    string `json:"reason,omitempty"`     // Dequeue reason, only for dequeue/abandoned/timeout.
	WaitTime  int64  `json:"wait_time,omitempty"`  // Wait time in seconds, only for dequeue/abandoned/timeout.
}

// Event topic constants.
const (
	TopicCallEnqueue = "vocara.acd.call.enqueue"
	TopicCallResume  = "vocara.acd.call.resume"
	TopicCallDequeue = "vocara.acd.call.dequeue"
	TopicAgentReady  = "vocara.agent.ready" // Agent ready event published by agentcenter.
)

func traceIDFromContext(ctx context.Context) string {
	if traceID, ok := ctx.Value(logger.TraceIDKey).(string); ok {
		return traceID
	}
	return ""
}

func (s *acdService) publishEnqueueEvent(ctx context.Context, call *model.CallInfo) {
	event := CallEnqueueEvent{
		CallID:       call.CallID,
		QueueID:      call.QueueID,
		SkillGroupID: call.SkillGroupID,
		Priority:     call.Priority,
		TenantID:     call.TenantID,
		Timestamp:    time.Now().Unix(),
		TraceID:      traceIDFromContext(ctx),
	}

	if err := s.eventBus.Publish(ctx, TopicCallEnqueue, event); err != nil {
		logger.WarnCtx(ctx, "publish enqueue event failed",
			zap.String("callId", call.CallID),
			zap.Error(err),
		)
	}

	// Publish to the unified event bus for WebSocket push and other consumers.
	s.publishBusEvent(ctx, eventbus.EventQueueEnqueue, call.TenantID, QueueEventData{
		CallID:    call.CallID,
		QueueID:   call.QueueID,
		TenantID:  call.TenantID,
		CallerNum: call.Caller,
		CalledNum: call.Called,
		Priority:  int(call.Priority),
		Position:  0, // The enqueue position can be calculated separately.
	})
}

func (s *acdService) publishResumeEvent(ctx context.Context, call *model.CallInfo) {
	event := CallResumeEvent{
		CallID:    call.CallID,
		QueueID:   call.QueueID,
		TenantID:  call.TenantID,
		Timestamp: time.Now().Unix(),
		TraceID:   traceIDFromContext(ctx),
	}

	if err := s.eventBus.Publish(ctx, TopicCallResume, event); err != nil {
		logger.WarnCtx(ctx, "publish resume event failed",
			zap.String("callId", call.CallID),
			zap.Error(err),
		)
	}
}

func (s *acdService) publishDequeueEvent(ctx context.Context, call *model.CallInfo, reason string, waitTime int64) {
	event := CallDequeueEvent{
		CallID:    call.CallID,
		QueueID:   call.QueueID,
		Reason:    reason,
		WaitTime:  waitTime,
		Timestamp: time.Now().Unix(),
		TraceID:   traceIDFromContext(ctx),
	}

	if err := s.eventBus.Publish(ctx, TopicCallDequeue, event); err != nil {
		logger.WarnCtx(ctx, "publish dequeue event failed",
			zap.String("callId", call.CallID),
			zap.Error(err),
		)
	}

	// Publish to the unified event bus for WebSocket push and other consumers.
	busEventType := eventbus.EventQueueDequeue
	if reason == "abandoned" {
		busEventType = eventbus.EventQueueAbandoned
	} else if reason == "timeout" {
		busEventType = eventbus.EventQueueTimeout
	}
	s.publishBusEvent(ctx, busEventType, call.TenantID, QueueEventData{
		CallID:    call.CallID,
		QueueID:   call.QueueID,
		TenantID:  call.TenantID,
		CallerNum: call.Caller,
		CalledNum: call.Called,
		Priority:  int(call.Priority),
		Reason:    reason,
		WaitTime:  waitTime,
	})
}

// publishBusEvent publishes asynchronously to the unified event bus.
// It is non-blocking; failures are logged without affecting the main flow.
func (s *acdService) publishBusEvent(ctx context.Context, eventType string, tenantID int64, data interface{}) {
	if s.busPublisher == nil {
		return
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.WarnCtx(ctx, "publish bus event panic recovered",
					zap.String("eventType", eventType),
					zap.Any("recover", r),
				)
			}
		}()

		event := eventbus.NewEvent(eventType, uint64(tenantID), eventSourceACD, data)
		if err := s.busPublisher.Publish(ctx, event); err != nil {
			logger.WarnCtx(ctx, "publish bus event failed",
				zap.String("eventType", eventType),
				zap.Error(err),
			)
		}
	}()
}

// --- Error mapping ---

func (s *acdService) mapLuaError(err error) error {
	errMsg := err.Error()
	switch {
	case contains(errMsg, "CALL_ALREADY_EXISTS"):
		return errors.NewBizError(errors.CodeConflict, "call already in queue")
	case contains(errMsg, "QUEUE_FULL"):
		return errors.NewBizError(errors.CodeQueueFull, "queue is full")
	case contains(errMsg, "CALL_NOT_FOUND"):
		return errors.NewBizError(errors.CodeNotFound, "call not found")
	case contains(errMsg, "ALREADY_PAUSED"):
		return errors.NewBizError(errors.CodeConflict, "call already paused")
	case contains(errMsg, "ALREADY_ACTIVE"):
		return errors.NewBizError(errors.CodeConflict, "call already active")
	case contains(errMsg, "INVALID_STATUS"):
		return errors.NewBizError(errors.CodeParamError, "invalid call status")
	case contains(errMsg, "NOT_IN_ACTIVE_QUEUE"):
		return errors.NewBizError(errors.CodeParamError, "call not in active queue")
	case contains(errMsg, "NOT_IN_PAUSED_QUEUE"):
		return errors.NewBizError(errors.CodeParamError, "call not in paused queue")
	default:
		return errors.NewBizError(errors.CodeSystemError, "acd operation failed")
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
