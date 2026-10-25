// Package service 提供 ACD 核心业务逻辑
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

// AcdService ACD 服务接口
type AcdService interface {
	// Enqueue 通话入队
	Enqueue(ctx context.Context, req *EnqueueRequest) (*model.CallInfo, error)
	// Dequeue 通话出队（挂机、转接、分配成功等）
	Dequeue(ctx context.Context, callID string, reason string) error
	// Pause 暂停通话分配（进入 IVR 等场景）
	Pause(ctx context.Context, callID string, reason string) error
	// Resume 恢复通话分配
	Resume(ctx context.Context, callID string) error
	// GetCallInfo 查询通话信息
	GetCallInfo(ctx context.Context, callID string) (*model.CallInfo, error)
}

// EnqueueRequest 入队请求
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

// acdService ACD 服务实现
type acdService struct {
	repo        *repository.AcdRedisRepo
	statsRepo   *repository.StatsRepo
	eventBus    EventPublisher   // 内部调度用的事件总线
	busPublisher eventbus.Publisher // 统一事件总线（用于 WebSocket 推送等）
	acdConfig   ACDConfig
}

const eventSourceACD = "acd"

// ACDConfig ACD 配置
type ACDConfig struct {
	// MaxQueueLen 最大排队长度，0 表示不限制
	MaxQueueLen int `json:"maxQueueLen"`
	// MaxPriority 最大优先级
	MaxPriority int `json:"maxPriority"`
	// PriorityAgingCycle 优先级老化周期（秒）
	PriorityAgingCycle int `json:"priorityAgingCycle"`
	// MaxWaitTime 最大等待时间（秒），超时自动转接或排队失败
	MaxWaitTime int `json:"maxWaitTime"`
}

// EventPublisher 事件发布器接口
type EventPublisher interface {
	// Publish 发布事件
	Publish(ctx context.Context, topic string, event interface{}) error
}

// redisEventPublisher Redis Pub/Sub 事件发布器
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

// NewAcdService 创建 ACD 服务
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

// NewRedisEventPublisher 创建 Redis 事件发布器
func NewRedisEventPublisher(repo *repository.AcdRedisRepo) EventPublisher {
	return &redisEventPublisher{repo: repo}
}

// Enqueue 通话入队
func (s *acdService) Enqueue(ctx context.Context, req *EnqueueRequest) (*model.CallInfo, error) {
	// 参数校验
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

	// 构造通话信息
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

	// 原子入队
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

	// 更新统计
	if err := s.statsRepo.IncrTotalToday(ctx, req.QueueID); err != nil {
		logger.WarnCtx(ctx, "increment total today failed",
			zap.Int64("queueId", req.QueueID),
			zap.Error(err),
		)
	}

	// 发布入队事件，触发调度器立即分配
	s.publishEnqueueEvent(ctx, call)

	return call, nil
}

// validDequeueReasons 出队原因白名单，与 dequeue.lua 的 reason 取值一致
var validDequeueReasons = map[string]bool{
	"answered": true,
	"abandoned": true,
	"transferred": true,
	"timeout": true,
	"error": true,
}

// Dequeue 通话出队
func (s *acdService) Dequeue(ctx context.Context, callID string, reason string) error {
	if callID == "" {
		return errors.NewBizError(errors.CodeParamError, "callId is required")
	}
	if !validDequeueReasons[reason] {
		return errors.NewBizError(errors.CodeParamError, "invalid dequeue reason: "+reason)
	}

	// 先获取通话信息（需要 queueID）
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

	// 原子出队
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

	// 发布出队事件
	s.publishDequeueEvent(ctx, call, reason, result.WaitTime)

	return nil
}

// Pause 暂停通话分配
func (s *acdService) Pause(ctx context.Context, callID string, reason string) error {
	if callID == "" {
		return errors.NewBizError(errors.CodeParamError, "callId is required")
	}

	// 获取通话信息
	call, err := s.repo.GetCallInfo(ctx, callID)
	if err != nil {
		return errors.NewBizError(errors.CodeSystemError, "get call info failed")
	}
	if call == nil {
		return errors.NewBizError(errors.CodeNotFound, "call not found: "+callID)
	}

	// 原子暂停
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

// Resume 恢复通话分配
func (s *acdService) Resume(ctx context.Context, callID string) error {
	if callID == "" {
		return errors.NewBizError(errors.CodeParamError, "callId is required")
	}

	// 获取通话信息
	call, err := s.repo.GetCallInfo(ctx, callID)
	if err != nil {
		return errors.NewBizError(errors.CodeSystemError, "get call info failed")
	}
	if call == nil {
		return errors.NewBizError(errors.CodeNotFound, "call not found: "+callID)
	}

	// 原子恢复
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

	// 发布恢复事件，触发调度器分配
	s.publishResumeEvent(ctx, call)

	return nil
}

// GetCallInfo 查询通话信息
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

// --- 事件发布 ---

// CallEnqueueEvent 入队事件
type CallEnqueueEvent struct {
	CallID       string         `json:"callId"`
	QueueID      int64          `json:"queueId"`
	SkillGroupID int64          `json:"skillGroupId"`
	Priority     model.Priority `json:"priority"`
	TenantID     int64          `json:"tenantId"`
	Timestamp    int64          `json:"timestamp"`
	TraceID      string         `json:"traceId,omitempty"`
}

// CallResumeEvent 恢复事件
type CallResumeEvent struct {
	CallID    string `json:"callId"`
	QueueID   int64  `json:"queueId"`
	TenantID  int64  `json:"tenantId"`
	Timestamp int64  `json:"timestamp"`
	TraceID   string `json:"traceId,omitempty"`
}

// CallDequeueEvent 出队事件
type CallDequeueEvent struct {
	CallID    string `json:"callId"`
	QueueID   int64  `json:"queueId"`
	Reason    string `json:"reason"`
	WaitTime  int64  `json:"waitTime"`
	Timestamp int64  `json:"timestamp"`
	TraceID   string `json:"traceId,omitempty"`
}

// QueueEventData 统一队列事件数据结构（发布到统一事件总线）
// 字段与 push 服务的 CallQueueEventData 对应
type QueueEventData struct {
	CallID    string `json:"call_id"`              // 通话ID
	QueueID   int64  `json:"queue_id"`             // 队列ID
	TenantID  int64  `json:"tenant_id"`            // 租户ID
	CallerNum string `json:"caller_num"`           // 主叫号码
	CalledNum string `json:"called_num"`           // 被叫号码
	Priority  int    `json:"priority"`             // 优先级
	Position  int    `json:"position"`             // 排队位置
	Reason    string `json:"reason,omitempty"`     // 出队原因（仅 dequeue/abandoned/timeout）
	WaitTime  int64  `json:"wait_time,omitempty"`  // 等待时间秒数（仅 dequeue/abandoned/timeout）
}

// 事件主题常量
const (
	TopicCallEnqueue = "vocara.acd.call.enqueue"
	TopicCallResume  = "vocara.acd.call.resume"
	TopicCallDequeue = "vocara.acd.call.dequeue"
	TopicAgentReady  = "vocara.agent.ready" // 座席就绪事件（由 agentcenter 发布）
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

	// 发布到统一事件总线（用于 WebSocket 推送等）
	s.publishBusEvent(ctx, eventbus.EventQueueEnqueue, call.TenantID, QueueEventData{
		CallID:    call.CallID,
		QueueID:   call.QueueID,
		TenantID:  call.TenantID,
		CallerNum: call.Caller,
		CalledNum: call.Called,
		Priority:  int(call.Priority),
		Position:  0, // 入队时的位置可以后续计算
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

	// 发布到统一事件总线（用于 WebSocket 推送等）
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

// publishBusEvent 异步发布到统一事件总线
// 非阻塞，失败仅打日志，不影响主流程
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

// --- 错误映射 ---

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
