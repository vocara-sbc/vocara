package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"vocara/contracts/event"
	"vocara/pkg/eventbus"
	"vocara/pkg/logger"
	"vocara/services/acd/internal/repository"
)

// Scheduler ACD 事件驱动调度器
// 负责监听事件并触发通话分配
type Scheduler struct {
	repo        *repository.AcdRedisRepo
	eventBus    EventPublisher
	busPublisher eventbus.Publisher // 统一事件总线（分配成功通知 callctrl 振铃）
	agentClient AgentCenterClient
	acdConfig   ACDConfig

	// 调度配置
	scheduleInterval   time.Duration // 定时兜底扫描间隔
	lockTimeout        time.Duration // 分布式锁超时时间
	allocateMaxRetries int           // 分配最大重试次数

	// 控制
	mu        sync.RWMutex
	started   bool
	stopCh    chan struct{}
	triggerCh chan int64 // 触发分配的队列 ID 通道（带限流）
}

// SchedulerConfig 调度器配置
type SchedulerConfig struct {
	// ScheduleInterval 定时兜底扫描间隔（默认 5 秒）
	ScheduleInterval time.Duration
	// LockTimeout 分布式锁超时时间（默认 10 秒）
	LockTimeout time.Duration
	// AllocateMaxRetries 分配失败最大重试次数（默认 3 次）
	AllocateMaxRetries int
}

// DefaultSchedulerConfig 默认调度器配置
func DefaultSchedulerConfig() SchedulerConfig {
	return SchedulerConfig{
		ScheduleInterval:   5 * time.Second,
		LockTimeout:        10 * time.Second,
		AllocateMaxRetries: 3,
	}
}

// NewScheduler 创建调度器
func NewScheduler(
	repo *repository.AcdRedisRepo,
	eventBus EventPublisher,
	agentClient AgentCenterClient,
	config ACDConfig,
	schedConfig SchedulerConfig,
	busPublishers ...eventbus.Publisher,
) *Scheduler {
	if schedConfig.ScheduleInterval <= 0 {
		schedConfig.ScheduleInterval = 5 * time.Second
	}
	if schedConfig.LockTimeout <= 0 {
		schedConfig.LockTimeout = 10 * time.Second
	}
	if schedConfig.AllocateMaxRetries <= 0 {
		schedConfig.AllocateMaxRetries = 3
	}

	var busPub eventbus.Publisher
	if len(busPublishers) > 0 {
		busPub = busPublishers[0]
	}

	return &Scheduler{
		repo:               repo,
		eventBus:           eventBus,
		busPublisher:       busPub,
		agentClient:        agentClient,
		acdConfig:          config,
		scheduleInterval:   schedConfig.ScheduleInterval,
		lockTimeout:        schedConfig.LockTimeout,
		allocateMaxRetries: schedConfig.AllocateMaxRetries,
		stopCh:             make(chan struct{}),
		triggerCh:          make(chan int64, 1000), // 缓冲 1000 个触发事件
	}
}

// Start 启动调度器
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return fmt.Errorf("scheduler already started")
	}
	s.started = true
	s.mu.Unlock()

	logger.InfoCtx(ctx, "acd scheduler starting",
		zap.Duration("scheduleInterval", s.scheduleInterval),
		zap.Duration("lockTimeout", s.lockTimeout),
		zap.Int("maxPriority", s.acdConfig.MaxPriority),
		zap.Int("priorityAgingCycle", s.acdConfig.PriorityAgingCycle),
		zap.Int("maxWaitTime", s.acdConfig.MaxWaitTime),
	)

	// 1. 启动事件订阅（Redis Pub/Sub）
	go s.subscribeEvents(ctx)

	// 2. 启动触发处理协程（合并事件，去重）
	go s.processTriggers(ctx)

	// 3. 启动定时兜底扫描
	go s.scheduleLoop(ctx)

	logger.InfoCtx(ctx, "acd scheduler started")
	return nil
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return
	}
	s.started = false
	close(s.stopCh)
}

// TriggerQueue 手动触发某个队列的分配
func (s *Scheduler) TriggerQueue(queueID int64) {
	select {
	case s.triggerCh <- queueID:
	default:
		// 通道满了就丢弃，反正有定时兜底
	}
}

// --- 事件订阅 ---

// subscribeEvents 订阅事件
func (s *Scheduler) subscribeEvents(ctx context.Context) {
	// 订阅的频道
	channels := []string{
		TopicCallEnqueue,
		TopicCallResume,
		TopicAgentReady,
	}

	pubsub := s.repo.Subscribe(ctx, channels...)
	defer pubsub.Close()

	logger.InfoCtx(ctx, "subscribed to acd events",
		zap.Strings("channels", channels),
	)

	ch := pubsub.Channel()
	for {
		select {
		case <-s.stopCh:
			return
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				logger.WarnCtx(ctx, "event channel closed")
				return
			}
			s.handleEvent(ctx, msg.Channel, msg.Payload)
		}
	}
}

// handleEvent 处理事件
func (s *Scheduler) handleEvent(ctx context.Context, channel string, payload string) {
	switch channel {
	case TopicCallEnqueue:
		var event CallEnqueueEvent
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			logger.WarnCtx(ctx, "failed to unmarshal enqueue event", zap.Error(err))
			return
		}
		logger.DebugCtx(ctx, "received enqueue event",
			zap.String("callId", event.CallID),
			zap.Int64("queueId", event.QueueID),
		)
		s.TriggerQueue(event.QueueID)

	case TopicCallResume:
		var event CallResumeEvent
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			logger.WarnCtx(ctx, "failed to unmarshal resume event", zap.Error(err))
			return
		}
		logger.DebugCtx(ctx, "received resume event",
			zap.String("callId", event.CallID),
			zap.Int64("queueId", event.QueueID),
		)
		s.TriggerQueue(event.QueueID)

	case TopicAgentReady:
		// 座席就绪事件，需要根据技能组找到对应的队列
		var event struct {
			AgentID      string `json:"agentId"`
			SkillGroupID string `json:"skillGroupId"`
			TenantID     string `json:"tenantId"`
			QueueID      int64  `json:"queueId"` // 可选：直接指定队列
		}
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			logger.WarnCtx(ctx, "failed to unmarshal agent ready event", zap.Error(err))
			return
		}
		logger.DebugCtx(ctx, "received agent ready event",
			zap.String("agentId", event.AgentID),
			zap.String("skillGroupId", event.SkillGroupID),
		)

		// 如果事件里直接带了 queueId，直接触发
		if event.QueueID > 0 {
			s.TriggerQueue(event.QueueID)
		}
		// 注：技能组到队列的映射需要通过配置获取，这里简化处理
		// 实际生产中应该从配置中心或数据库获取技能组关联的所有队列
		// 目前通过事件携带 queueId 来处理
	}
}

// --- 触发处理（合并 + 去重）---

// processTriggers 处理触发事件，合并短时间内多次触发为一次分配
func (s *Scheduler) processTriggers(ctx context.Context) {
	// 使用 map 去重，定期批量处理
	type triggerInfo struct {
		count int
	}
	pending := make(map[int64]*triggerInfo)
	ticker := time.NewTicker(100 * time.Millisecond) // 每 100ms 处理一批
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ctx.Done():
			return

		case queueID := <-s.triggerCh:
			if _, ok := pending[queueID]; ok {
				pending[queueID].count++
			} else {
				pending[queueID] = &triggerInfo{count: 1}
			}

		case <-ticker.C:
			if len(pending) == 0 {
				continue
			}

			// 取出所有待处理的队列
			queues := make([]int64, 0, len(pending))
			for qid := range pending {
				queues = append(queues, qid)
			}
			pending = make(map[int64]*triggerInfo)

			// 并发分配（每个队列一个 goroutine）
			for _, qid := range queues {
				go s.safeAllocateQueue(ctx, qid)
			}

			logger.DebugCtx(ctx, "batch allocate triggered",
				zap.Int("queueCount", len(queues)),
			)
		}
	}
}

// safeAllocateQueue 安全地执行队列分配（带 recover）
func (s *Scheduler) safeAllocateQueue(ctx context.Context, queueID int64) {
	defer func() {
		if r := recover(); r != nil {
			logger.ErrorCtx(ctx, "allocate queue panic",
				zap.Int64("queueId", queueID),
				zap.Any("panic", r),
			)
		}
	}()

	s.allocateQueue(ctx, queueID)
}

// --- 定时兜底 ---

// scheduleLoop 定时兜底扫描
func (s *Scheduler) scheduleLoop(ctx context.Context) {
	ticker := time.NewTicker(s.scheduleInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.scanAndAllocate(ctx)
		}
	}
}

// scanAndAllocate 扫描所有工作中的队列并触发分配
func (s *Scheduler) scanAndAllocate(ctx context.Context) {
	// 获取所有工作中的队列
	queueIDs, err := s.repo.GetWorkingQueueIDs(ctx)
	if err != nil {
		logger.ErrorCtx(ctx, "get working queue ids failed", zap.Error(err))
		return
	}

	if len(queueIDs) == 0 {
		return
	}

	logger.DebugCtx(ctx, "scheduler scan",
		zap.Int("workingQueueCount", len(queueIDs)),
	)

	// 触发每个队列的分配
	for _, qid := range queueIDs {
		s.TriggerQueue(qid)
	}
}

// --- 核心分配逻辑 ---

// allocateQueue 分配一个队列的通话
// 1. 尝试获取分布式锁
// 2. 循环取队首 → 问座席 → 成功继续，失败退出
func (s *Scheduler) allocateQueue(ctx context.Context, queueID int64) {
	// 1. 尝试获取分布式锁
	locked, err := s.repo.TryLock(ctx, queueID, s.lockTimeout)
	if err != nil {
		logger.ErrorCtx(ctx, "try lock failed",
			zap.Int64("queueId", queueID),
			zap.Error(err),
		)
		return
	}
	if !locked {
		// 其他节点正在分配，直接返回
		logger.DebugCtx(ctx, "skip allocate: another node holds the lock",
			zap.Int64("queueId", queueID),
		)
		return
	}
	defer func() {
		if err := s.repo.Unlock(ctx, queueID); err != nil {
			logger.WarnCtx(ctx, "unlock failed",
				zap.Int64("queueId", queueID),
				zap.Error(err),
			)
		}
	}()

	// 2. 分配循环
	allocatedCount := 0
	var skillGroupID string
	var tenantID string

	for i := 0; i < 100; i++ { // 单次最多分配 100 个，防止锁超时
		// 2.1 从队首取一个通话
		agingCycle := s.acdConfig.PriorityAgingCycle
		maxPriority := s.acdConfig.MaxPriority
		if maxPriority <= 0 {
			maxPriority = 255
		}
		call, err := s.repo.Allocate(ctx, queueID, agingCycle, maxPriority)
		if err != nil {
			// 检查是否是优先级更新重试的情况
			if contains(err.Error(), "PRIORITY_UPDATED_RETRY") {
				logger.DebugCtx(ctx, "priority updated, retry allocate",
					zap.Int64("queueId", queueID),
				)
				continue
			}
			logger.ErrorCtx(ctx, "allocate call from queue failed",
				zap.Int64("queueId", queueID),
				zap.Error(err),
			)
			break
		}
		if call == nil {
			// 队列为空
			logger.DebugCtx(ctx, "queue is empty, stop allocate",
				zap.Int64("queueId", queueID),
			)
			break
		}

		// 2.2 从通话信息中获取技能组ID和租户ID（仅第一次需要）
		// 注意：不使用 queueID 代替 skillGroupID，二者可能不同
		if skillGroupID == "" {
			skillGroupID = fmt.Sprintf("%d", call.SkillGroupID)
		}
		if tenantID == "" {
			tenantID = fmt.Sprintf("%d", call.TenantID)
			if tenantID == "0" {
				tenantID = "1" // 兜底：默认租户
			}
		}

		// 2.3 向座席中心请求空闲座席
		agentID, dnNumber, err := s.agentClient.GetReadyAgent(ctx, tenantID, skillGroupID)
		if err != nil {
			logger.ErrorCtx(ctx, "get ready agent failed",
				zap.String("callId", call.CallID),
				zap.String("skillGroupId", skillGroupID),
				zap.Error(err),
			)
			// 座席中心异常，本次分配终止
			// 注意：Allocate 是 peek 语义（ZRANGE，不从队列移除），
			// 所以分配失败时不需要放回，通话仍在 active 队列中
			break
		}
		if agentID == "" {
			// 没有空闲座席
			logger.DebugCtx(ctx, "no ready agent, stop allocate",
				zap.String("callId", call.CallID),
				zap.String("skillGroupId", skillGroupID),
			)
			// 注意：Allocate 是 peek 语义，不需要放回操作
			break
		}

		// 2.4 尝试分配给座席（通知振铃）
		assignErr := s.agentClient.AssignCall(ctx, tenantID, agentID, call.CallID, call)
		if assignErr != nil {
			logger.WarnCtx(ctx, "assign call to agent failed",
				zap.String("callId", call.CallID),
				zap.String("agentId", agentID),
				zap.Error(assignErr),
			)
			// 分配失败，座席可能被其他节点抢占了，重试下一个座席
			// 先恢复座席为就绪状态
			_ = s.agentClient.ResumeReady(ctx, tenantID, agentID)
			continue
		}

		// 2.5 分配成功，按已应答出队
		_, dequeueErr := s.repo.Dequeue(ctx, call.CallID, queueID, "answered")
		if dequeueErr != nil {
			logger.ErrorCtx(ctx, "dequeue after allocate failed",
				zap.String("callId", call.CallID),
				zap.String("agentId", agentID),
				zap.Error(dequeueErr),
			)
			// 出队失败但已经分配给座席了，需要告警或补偿
			// 这里继续尝试下一个
			continue
		}

		allocatedCount++
		logger.InfoCtx(ctx, "call allocated",
			zap.String("callId", call.CallID),
			zap.String("agentId", agentID),
			zap.String("dnNumber", dnNumber),
			zap.Int("priority", int(call.CurrentPriority)),
			zap.Int64("waitSeconds", call.WaitSeconds),
		)

		// 发布分配成功事件，callctrl 收到后对座席分机发起振铃并桥接客户腿
		s.publishAllocatedEvent(ctx, call, queueID, agentID, dnNumber)
	}

	if allocatedCount > 0 {
		logger.InfoCtx(ctx, "allocate round finished",
			zap.Int64("queueId", queueID),
			zap.Int("allocatedCount", allocatedCount),
		)
	}
}

// publishAllocatedEvent 通过统一事件总线发布分配成功事件。
// 主题 queue.dequeue，callctrl 订阅后执行 originate 座席分机。
func (s *Scheduler) publishAllocatedEvent(ctx context.Context, call *repository.AllocateResult,
	queueID int64, agentID, dnNumber string) {
	if s.busPublisher == nil {
		logger.WarnCtx(ctx, "bus publisher nil, skip allocated event",
			zap.String("callId", call.CallID))
		return
	}

	tenantID := int64(1)
	if call.TenantID > 0 {
		tenantID = call.TenantID
	}

	data := event.AllocatedCall{
		CallID:       call.CallID,
		QueueID:      queueID,
		SkillGroupID: call.SkillGroupID,
		TenantID:     tenantID,
		AgentID:      agentID,
		DnNumber:     dnNumber,
		Caller:       call.Caller,
		Called:       call.Called,
	}

	go func() {
		evCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		ev := eventbus.NewEvent(eventbus.EventQueueDequeue, uint64(tenantID), "acd", data)
		if err := s.busPublisher.Publish(evCtx, ev); err != nil {
			logger.WarnCtx(evCtx, "publish allocated event failed",
				zap.String("callId", call.CallID),
				zap.Error(err))
		}
	}()
}
