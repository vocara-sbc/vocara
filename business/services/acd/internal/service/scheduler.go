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

// Scheduler is the event-driven ACD scheduler.
// It listens for events and triggers call allocation.
type Scheduler struct {
	repo        *repository.AcdRedisRepo
	eventBus    EventPublisher
	busPublisher eventbus.Publisher // Unified event bus for notifying callctrl to ring after successful allocation.
	agentClient AgentCenterClient
	acdConfig   ACDConfig

	// Scheduling configuration.
	scheduleInterval   time.Duration // Fallback scan interval.
	lockTimeout        time.Duration // Distributed lock timeout.
	allocateMaxRetries int           // Maximum allocation retries.

	// Control state.
	mu        sync.RWMutex
	started   bool
	stopCh    chan struct{}
	triggerCh chan int64 // Queue ID channel that triggers allocation, with throttling.
}

// SchedulerConfig contains scheduler configuration.
type SchedulerConfig struct {
	// ScheduleInterval is the fallback scan interval, defaulting to 5 seconds.
	ScheduleInterval time.Duration
	// LockTimeout is the distributed lock timeout, defaulting to 10 seconds.
	LockTimeout time.Duration
	// AllocateMaxRetries is the maximum number of allocation retries, defaulting to 3.
	AllocateMaxRetries int
}

// DefaultSchedulerConfig returns the default scheduler configuration.
func DefaultSchedulerConfig() SchedulerConfig {
	return SchedulerConfig{
		ScheduleInterval:   5 * time.Second,
		LockTimeout:        10 * time.Second,
		AllocateMaxRetries: 3,
	}
}

// NewScheduler creates a scheduler.
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
		triggerCh:          make(chan int64, 1000), // Buffer up to 1000 trigger events.
	}
}

// Start starts the scheduler.
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

	// 1. Start the event subscription over Redis Pub/Sub.
	go s.subscribeEvents(ctx)

	// 2. Start the trigger-processing goroutine to coalesce and deduplicate events.
	go s.processTriggers(ctx)

	// 3. Start the periodic fallback scan.
	go s.scheduleLoop(ctx)

	logger.InfoCtx(ctx, "acd scheduler started")
	return nil
}

// Stop stops the scheduler.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return
	}
	s.started = false
	close(s.stopCh)
}

// TriggerQueue manually triggers allocation for a queue.
func (s *Scheduler) TriggerQueue(queueID int64) {
	select {
	case s.triggerCh <- queueID:
	default:
		// Drop the trigger when the channel is full; the periodic fallback scan still runs.
	}
}

// --- Event subscription ---

// subscribeEvents subscribes to events.
func (s *Scheduler) subscribeEvents(ctx context.Context) {
	// Channels to subscribe to.
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

// handleEvent handles an event.
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
		// Agent ready events require finding the corresponding queue from the skill group.
		var event struct {
			AgentID      string `json:"agentId"`
			SkillGroupID string `json:"skillGroupId"`
			TenantID     string `json:"tenantId"`
			QueueID      int64  `json:"queueId"` // Optional: specify the queue directly.
		}
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			logger.WarnCtx(ctx, "failed to unmarshal agent ready event", zap.Error(err))
			return
		}
		logger.DebugCtx(ctx, "received agent ready event",
			zap.String("agentId", event.AgentID),
			zap.String("skillGroupId", event.SkillGroupID),
		)

		// Trigger directly when the event contains queueId.
		if event.QueueID > 0 {
			s.TriggerQueue(event.QueueID)
		}
		// Note: skill-group-to-queue mappings come from configuration; this path uses a simplified approach.
		// Production deployments should obtain all queues associated with a skill group from a configuration center or database.
		// This path handles allocation through the queueId carried by the event.
	}
}

// --- Trigger processing (coalescing + deduplication) ---

// processTriggers handles trigger events and coalesces multiple triggers within a short interval into one allocation.
func (s *Scheduler) processTriggers(ctx context.Context) {
	// Use a map for deduplication and process batches periodically.
	type triggerInfo struct {
		count int
	}
	pending := make(map[int64]*triggerInfo)
	ticker := time.NewTicker(100 * time.Millisecond) // Process one batch every 100ms.
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

			// Take all pending queues.
			queues := make([]int64, 0, len(pending))
			for qid := range pending {
				queues = append(queues, qid)
			}
			pending = make(map[int64]*triggerInfo)

			// Allocate concurrently, with one goroutine per queue.
			for _, qid := range queues {
				go s.safeAllocateQueue(ctx, qid)
			}

			logger.DebugCtx(ctx, "batch allocate triggered",
				zap.Int("queueCount", len(queues)),
			)
		}
	}
}

// safeAllocateQueue executes queue allocation with panic recovery.
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

// --- Periodic fallback ---

// scheduleLoop runs the periodic fallback scan.
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

// scanAndAllocate scans all working queues and triggers allocation.
func (s *Scheduler) scanAndAllocate(ctx context.Context) {
	// Get all working queues.
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

	// Trigger allocation for each queue.
	for _, qid := range queueIDs {
		s.TriggerQueue(qid)
	}
}

// --- Core allocation logic ---

// allocateQueue allocates calls from one queue.
// 1. Attempt to acquire the distributed lock.
// 2. Repeatedly peek at the queue head, request an agent, continue on success, and exit on failure.
func (s *Scheduler) allocateQueue(ctx context.Context, queueID int64) {
	// 1. Attempt to acquire the distributed lock.
	locked, err := s.repo.TryLock(ctx, queueID, s.lockTimeout)
	if err != nil {
		logger.ErrorCtx(ctx, "try lock failed",
			zap.Int64("queueId", queueID),
			zap.Error(err),
		)
		return
	}
	if !locked {
		// Another node is allocating; return directly.
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

	// 2. Allocation loop.
	allocatedCount := 0
	var skillGroupID string
	var tenantID string

	for i := 0; i < 100; i++ { // Allocate at most 100 calls per round to prevent lock timeout.
		// 2.1 Peek at one call from the queue head.
		agingCycle := s.acdConfig.PriorityAgingCycle
		maxPriority := s.acdConfig.MaxPriority
		if maxPriority <= 0 {
			maxPriority = 255
		}
		call, err := s.repo.Allocate(ctx, queueID, agingCycle, maxPriority)
		if err != nil {
			// Check whether priority aging requires a retry.
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
			// The queue is empty.
			logger.DebugCtx(ctx, "queue is empty, stop allocate",
				zap.Int64("queueId", queueID),
			)
			break
		}

		// 2.2 Get the skill group ID and tenant ID from the call information; only the first call requires this.
		// Note: queueID cannot replace skillGroupID because they can differ.
		if skillGroupID == "" {
			skillGroupID = fmt.Sprintf("%d", call.SkillGroupID)
		}
		if tenantID == "" {
			tenantID = fmt.Sprintf("%d", call.TenantID)
			if tenantID == "0" {
				tenantID = "1" // Fallback to the default tenant.
			}
		}

		// 2.3 Request a ready agent from the agent center.
		agentID, dnNumber, err := s.agentClient.GetReadyAgent(ctx, tenantID, skillGroupID)
		if err != nil {
			logger.ErrorCtx(ctx, "get ready agent failed",
				zap.String("callId", call.CallID),
				zap.String("skillGroupId", skillGroupID),
				zap.Error(err),
			)
			// An agent center error terminates this allocation round.
			// Note: Allocate has peek semantics and uses ZRANGE without removing the call from the queue.
			// Failed allocation therefore requires no requeue operation; the call remains in the active queue.
			break
		}
		if agentID == "" {
			// No ready agent is available.
			logger.DebugCtx(ctx, "no ready agent, stop allocate",
				zap.String("callId", call.CallID),
				zap.String("skillGroupId", skillGroupID),
			)
			// Note: Allocate has peek semantics, so no requeue operation is required.
			break
		}

		// 2.4 Attempt to assign the call to the agent and trigger ringing.
		assignErr := s.agentClient.AssignCall(ctx, tenantID, agentID, call.CallID, call)
		if assignErr != nil {
			logger.WarnCtx(ctx, "assign call to agent failed",
				zap.String("callId", call.CallID),
				zap.String("agentId", agentID),
				zap.Error(assignErr),
			)
			// Allocation can fail when another node acquires the agent; retry with another agent.
			// Restore the agent to the ready state first.
			_ = s.agentClient.ResumeReady(ctx, tenantID, agentID)
			continue
		}

		// 2.5 After successful allocation, dequeue the call as answered.
		_, dequeueErr := s.repo.Dequeue(ctx, call.CallID, queueID, "answered")
		if dequeueErr != nil {
			logger.ErrorCtx(ctx, "dequeue after allocate failed",
				zap.String("callId", call.CallID),
				zap.String("agentId", agentID),
				zap.Error(dequeueErr),
			)
			// Dequeue failure after agent assignment requires an alert or compensation.
			// Continue with the next call.
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

		// Publish the allocation event; callctrl rings the agent extension and bridges the customer leg.
		s.publishAllocatedEvent(ctx, call, queueID, agentID, dnNumber)
	}

	if allocatedCount > 0 {
		logger.InfoCtx(ctx, "allocate round finished",
			zap.Int64("queueId", queueID),
			zap.Int("allocatedCount", allocatedCount),
		)
	}
}

// publishAllocatedEvent publishes the allocation event through the unified event bus.
// callctrl subscribes to queue.dequeue and originates a call to the agent extension.
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
