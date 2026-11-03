// Package repository provides the ACD service data access layer.
package repository

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"vocara/services/acd/internal/model"
	"vocara/services/acd/scripts"
)

// Redis key prefix constants.
const (
	keyPrefix = "vocara:acd:queue"

	// keyActive = vocara:acd:queue:active:{queueId}
	keyActive = "%s:active:%d"
	// keyPaused = vocara:acd:queue:paused:{queueId}
	keyPaused = "%s:paused:%d"
	// keyCall = vocara:acd:queue:call:{callId}
	keyCall = "%s:call:%s"
	// keyWorkingIDs = vocara:acd:queue:working-ids
	keyWorkingIDs = "%s:working-ids"
	// keyStats = vocara:acd:queue:stats:{queueId}
	keyStats = "%s:stats:%d"
	// keyLock = vocara:acd:queue:lock:{queueId}
	keyLock = "%s:lock:%d"
)

// Lua script content is embedded at compile time; see the vocara/services/acd/scripts package.

// AcdRedisRepo provides ACD Redis data access.
type AcdRedisRepo struct {
	client redis.UniversalClient

	// Precompiled Lua scripts.
	enqueueScript  *redis.Script
	allocateScript *redis.Script
	pauseScript    *redis.Script
	resumeScript   *redis.Script
	dequeueScript  *redis.Script
}

// NewAcdRedisRepo creates an ACD Redis repository.
func NewAcdRedisRepo(client redis.UniversalClient) *AcdRedisRepo {
	return &AcdRedisRepo{
		client:         client,
		enqueueScript:  redis.NewScript(scripts.Enqueue),
		allocateScript: redis.NewScript(scripts.Allocate),
		pauseScript:    redis.NewScript(scripts.Pause),
		resumeScript:   redis.NewScript(scripts.Resume),
		dequeueScript:  redis.NewScript(scripts.Dequeue),
	}
}

// --- Key constructors ---

func (r *AcdRedisRepo) activeKey(queueID int64) string {
	return fmt.Sprintf(keyActive, keyPrefix, queueID)
}

func (r *AcdRedisRepo) pausedKey(queueID int64) string {
	return fmt.Sprintf(keyPaused, keyPrefix, queueID)
}

func (r *AcdRedisRepo) callKey(callID string) string {
	return fmt.Sprintf(keyCall, keyPrefix, callID)
}

func (r *AcdRedisRepo) callKeyPrefix() string {
	return fmt.Sprintf(keyCall, keyPrefix, "")
}

func (r *AcdRedisRepo) workingIDsKey() string {
	return fmt.Sprintf(keyWorkingIDs, keyPrefix)
}

func (r *AcdRedisRepo) statsKey(queueID int64) string {
	return fmt.Sprintf(keyStats, keyPrefix, queueID)
}

func (r *AcdRedisRepo) lockKey(queueID int64) string {
	return fmt.Sprintf(keyLock, keyPrefix, queueID)
}

// --- Core operations ---

// EnqueueResult contains the enqueue result.
type EnqueueResult struct {
	Success bool
	Score   float64
}

// Enqueue atomically enqueues a call.
func (r *AcdRedisRepo) Enqueue(ctx context.Context, call *model.CallInfo, maxQueueLen int) (*EnqueueResult, error) {
	eqTimestamp := call.EQTime.Unix()

	// Calculate score: priority * 10^12 + eqTimestamp.
	score := float64(call.Priority)*1e12 + float64(eqTimestamp)

	keys := []string{
		r.activeKey(call.QueueID),
		r.callKey(call.CallID),
		r.workingIDsKey(),
	}

	argv := []interface{}{
		score,
		call.CallID,
		call.Caller,
		call.Called,
		strconv.FormatInt(call.QueueID, 10),
		strconv.FormatInt(call.SkillGroupID, 10),
		int(call.Priority),
		eqTimestamp,
		strconv.FormatInt(call.TenantID, 10),
		call.CustomerInfo,
		call.UUI,
		maxQueueLen,
	}

	res, err := r.enqueueScript.Run(ctx, r.client, keys, argv...).Result()
	if err != nil {
		return nil, fmt.Errorf("enqueue lua script failed: %w", err)
	}

	// Parse the return value.
	arr, ok := res.([]interface{})
	if !ok || len(arr) < 1 {
		return nil, fmt.Errorf("invalid enqueue response format")
	}

	status, _ := arr[0].(string)
	if status == "ok" || status == "ENQUEUED" {
		scoreVal := 0.0
		if len(arr) > 1 {
			scoreVal, _ = arr[1].(float64)
		}
		return &EnqueueResult{Success: true, Score: scoreVal}, nil
	}

	// Handle errors.
	errMsg := status
	return nil, fmt.Errorf("enqueue failed: %s", errMsg)
}

// AllocateResult contains the allocation result.
type AllocateResult struct {
	CallID          string
	Score           float64
	Caller          string
	Called          string
	QueueID         int64
	SkillGroupID    int64
	Priority        model.Priority
	CurrentPriority model.Priority
	EQTime          time.Time
	TenantID        int64
	CustomerInfo    string
	UUI             string
	WaitSeconds     int64
}

// Allocate peeks at the first call without deleting it; Dequeue removes it after successful allocation.
func (r *AcdRedisRepo) Allocate(ctx context.Context, queueID int64, agingCycle int, maxPriority int) (*AllocateResult, error) {
	now := time.Now().Unix()

	keys := []string{
		r.activeKey(queueID),
		r.pausedKey(queueID),
		r.callKeyPrefix(),
	}

	argv := []interface{}{
		now,
		agingCycle,
		maxPriority,
	}

	res, err := r.allocateScript.Run(ctx, r.client, keys, argv...).Result()
	if err != nil {
		// Check for an error returned by the script.
		if err.Error() == "EMPTY_QUEUE" {
			return nil, nil // an empty queue is not an error
		}
		return nil, fmt.Errorf("allocate lua script failed: %w", err)
	}

	arr, ok := res.([]interface{})
	if !ok || len(arr) < 1 {
		return nil, fmt.Errorf("invalid allocate response format")
	}

	status, _ := arr[0].(string)
	// The Lua script uses single-element arrays to represent an empty queue and a priority aging retry.
	if status == "EMPTY_QUEUE" {
		return nil, nil
	}
	if status == "PRIORITY_UPDATED_RETRY" {
		return nil, fmt.Errorf("PRIORITY_UPDATED_RETRY")
	}
	if status == "err" || status == "error" {
		errMsg := ""
		if len(arr) > 1 {
			errMsg, _ = arr[1].(string)
		}
		if errMsg == "EMPTY_QUEUE" {
			return nil, nil
		}
		return nil, fmt.Errorf("allocate failed: %s", errMsg)
	}

	// Parse the successful result, whose first element is "ok".
	if len(arr) < 14 {
		return nil, fmt.Errorf("allocate response too short: %d", len(arr))
	}

	callID, _ := arr[1].(string)
	score, _ := arr[2].(float64)
	caller, _ := arr[3].(string)
	called, _ := arr[4].(string)
	queueIDStr, _ := arr[5].(string)
	skillGroupIDStr, _ := arr[6].(string)
	priorityStr, _ := arr[7].(string)
	currentPriorityStr, _ := arr[8].(string)
	eqTimeStr, _ := arr[9].(string)
	tenantIDStr, _ := arr[10].(string)
	customerInfo, _ := arr[11].(string)
	uui, _ := arr[12].(string)
	waitSeconds, _ := arr[13].(int64)

	qid, _ := strconv.ParseInt(queueIDStr, 10, 64)
	sgid, _ := strconv.ParseInt(skillGroupIDStr, 10, 64)
	pri, _ := strconv.Atoi(priorityStr)
	cpri, _ := strconv.Atoi(currentPriorityStr)
	eqt, _ := strconv.ParseInt(eqTimeStr, 10, 64)
	tid, _ := strconv.ParseInt(tenantIDStr, 10, 64)

	return &AllocateResult{
		CallID:          callID,
		Score:           score,
		Caller:          caller,
		Called:          called,
		QueueID:         qid,
		SkillGroupID:    sgid,
		Priority:        model.Priority(pri),
		CurrentPriority: model.Priority(cpri),
		EQTime:          time.Unix(eqt, 0),
		TenantID:        tid,
		CustomerInfo:    customerInfo,
		UUI:             uui,
		WaitSeconds:     waitSeconds,
	}, nil
}

// Pause pauses call allocation.
func (r *AcdRedisRepo) Pause(ctx context.Context, callID string, queueID int64, reason string) error {
	pauseTime := time.Now().Unix()

	keys := []string{
		r.activeKey(queueID),
		r.pausedKey(queueID),
		r.callKey(callID),
	}

	argv := []interface{}{
		callID,
		pauseTime,
		reason,
	}

	_, err := r.pauseScript.Run(ctx, r.client, keys, argv...).Result()
	if err != nil {
		return fmt.Errorf("pause lua script failed: %w", err)
	}

	return nil
}

// ResumeResult contains the resume result.
type ResumeResult struct {
	Score float64
}

// Resume resumes call allocation.
func (r *AcdRedisRepo) Resume(ctx context.Context, callID string, queueID int64) (*ResumeResult, error) {
	resumeTime := time.Now().Unix()

	keys := []string{
		r.activeKey(queueID),
		r.pausedKey(queueID),
		r.callKey(callID),
		r.workingIDsKey(),
	}

	argv := []interface{}{
		callID,
		resumeTime,
	}

	res, err := r.resumeScript.Run(ctx, r.client, keys, argv...).Result()
	if err != nil {
		return nil, fmt.Errorf("resume lua script failed: %w", err)
	}

	arr, ok := res.([]interface{})
	if !ok || len(arr) < 1 {
		return nil, fmt.Errorf("invalid resume response format")
	}

	scoreVal := 0.0
	if len(arr) > 1 {
		scoreVal, _ = arr[1].(float64)
	}

	return &ResumeResult{Score: scoreVal}, nil
}

// DequeueResult contains the dequeue result.
type DequeueResult struct {
	WaitTime int64
}

// Dequeue removes a call from the queue.
func (r *AcdRedisRepo) Dequeue(ctx context.Context, callID string, queueID int64, reason string) (*DequeueResult, error) {
	dequeueTime := time.Now().Unix()

	keys := []string{
		r.activeKey(queueID),
		r.pausedKey(queueID),
		r.callKey(callID),
		r.statsKey(queueID),
		r.workingIDsKey(),
	}

	argv := []interface{}{
		callID,
		reason,
		dequeueTime,
	}

	res, err := r.dequeueScript.Run(ctx, r.client, keys, argv...).Result()
	if err != nil {
		return nil, fmt.Errorf("dequeue lua script failed: %w", err)
	}

	arr, ok := res.([]interface{})
	if !ok || len(arr) < 1 {
		return nil, fmt.Errorf("invalid dequeue response format")
	}

	waitTime := int64(0)
	if len(arr) > 1 {
		if wt, ok := arr[1].(int64); ok {
			waitTime = wt
		}
	}

	return &DequeueResult{WaitTime: waitTime}, nil
}

// --- Call information queries ---

// GetCallInfo returns call details.
func (r *AcdRedisRepo) GetCallInfo(ctx context.Context, callID string) (*model.CallInfo, error) {
	data, err := r.client.HGetAll(ctx, r.callKey(callID)).Result()
	if err != nil {
		return nil, fmt.Errorf("get call info failed: %w", err)
	}
	if len(data) == 0 {
		return nil, nil // not found
	}

	return hashToCallInfo(data), nil
}

// hashToCallInfo converts Redis HASH data to CallInfo.
func hashToCallInfo(data map[string]string) *model.CallInfo {
	call := &model.CallInfo{
		CallID:       data["callId"],
		Caller:       data["caller"],
		Called:       data["called"],
		CustomerInfo: data["customerInfo"],
		UUI:          data["uui"],
		Status:       model.CallStatus(data["status"]),
	}

	if v, err := strconv.ParseInt(data["queueId"], 10, 64); err == nil {
		call.QueueID = v
	}
	if v, err := strconv.ParseInt(data["skillGroupId"], 10, 64); err == nil {
		call.SkillGroupID = v
	}
	if v, err := strconv.Atoi(data["priority"]); err == nil {
		call.Priority = model.Priority(v)
	}
	if v, err := strconv.Atoi(data["currentPriority"]); err == nil {
		call.CurrentPriority = model.Priority(v)
	}
	if v, err := strconv.ParseInt(data["tenantId"], 10, 64); err == nil {
		call.TenantID = v
	}
	if v, err := strconv.ParseInt(data["eqTime"], 10, 64); err == nil {
		call.EQTime = time.Unix(v, 0)
	}
	if v, err := strconv.ParseInt(data["pauseTime"], 10, 64); err == nil {
		call.PauseTime = time.Unix(v, 0)
	}
	if v, err := strconv.ParseInt(data["allocateTime"], 10, 64); err == nil {
		call.AllocateTime = time.Unix(v, 0)
	}
	if v, err := strconv.ParseInt(data["agentId"], 10, 64); err == nil {
		call.AgentID = v
	}

	return call
}

// --- Queue length queries ---

// GetActiveQueueLen returns the active queue length.
func (r *AcdRedisRepo) GetActiveQueueLen(ctx context.Context, queueID int64) (int64, error) {
	return r.client.ZCard(ctx, r.activeKey(queueID)).Result()
}

// GetPausedQueueLen returns the paused queue length.
func (r *AcdRedisRepo) GetPausedQueueLen(ctx context.Context, queueID int64) (int64, error) {
	return r.client.ZCard(ctx, r.pausedKey(queueID)).Result()
}

// GetWorkingQueueIDs returns all working queue IDs.
func (r *AcdRedisRepo) GetWorkingQueueIDs(ctx context.Context) ([]int64, error) {
	members, err := r.client.SMembers(ctx, r.workingIDsKey()).Result()
	if err != nil {
		return nil, fmt.Errorf("get working queue ids failed: %w", err)
	}

	ids := make([]int64, 0, len(members))
	for _, m := range members {
		if id, err := strconv.ParseInt(m, 10, 64); err == nil {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// --- Distributed locks ---

// TryLock attempts to acquire a queue-level distributed lock.
func (r *AcdRedisRepo) TryLock(ctx context.Context, queueID int64, ttl time.Duration) (bool, error) {
	ok, err := r.client.SetNX(ctx, r.lockKey(queueID), "1", ttl).Result()
	if err != nil {
		return false, fmt.Errorf("try lock failed: %w", err)
	}
	return ok, nil
}

// Unlock releases a queue-level distributed lock.
func (r *AcdRedisRepo) Unlock(ctx context.Context, queueID int64) error {
	_, err := r.client.Del(ctx, r.lockKey(queueID)).Result()
	return err
}

// --- Pub/Sub ---

// Publish publishes an event.
func (r *AcdRedisRepo) Publish(ctx context.Context, channel string, message string) error {
	return r.client.Publish(ctx, channel, message).Err()
}

// Subscribe subscribes to events.
func (r *AcdRedisRepo) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return r.client.Subscribe(ctx, channels...)
}
