// Package repository 提供 ACD 服务的数据访问层
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

// Redis key 前缀常量
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

// Lua 脚本内容（编译时嵌入，见 vocara/services/acd/scripts 包）

// AcdRedisRepo ACD Redis 仓储
type AcdRedisRepo struct {
	client redis.UniversalClient

	// 预编译的 Lua 脚本
	enqueueScript  *redis.Script
	allocateScript *redis.Script
	pauseScript    *redis.Script
	resumeScript   *redis.Script
	dequeueScript  *redis.Script
}

// NewAcdRedisRepo 创建 ACD Redis 仓储
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

// --- Key 构造函数 ---

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

// --- 核心操作 ---

// EnqueueResult 入队结果
type EnqueueResult struct {
	Success bool
	Score   float64
}

// Enqueue 通话入队（原子操作）
func (r *AcdRedisRepo) Enqueue(ctx context.Context, call *model.CallInfo, maxQueueLen int) (*EnqueueResult, error) {
	eqTimestamp := call.EQTime.Unix()

	// 计算 score: priority * 10^12 + eqTimestamp
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

	// 解析返回值
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

	// 错误情况
	errMsg := status
	return nil, fmt.Errorf("enqueue failed: %s", errMsg)
}

// AllocateResult 分配结果
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

// Allocate 从队首取一个通话（不删除，分配成功后再调用 Dequeue）
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
		// 检查是否是脚本返回的错误
		if err.Error() == "EMPTY_QUEUE" {
			return nil, nil // 队列为空不是错误
		}
		return nil, fmt.Errorf("allocate lua script failed: %w", err)
	}

	arr, ok := res.([]interface{})
	if !ok || len(arr) < 1 {
		return nil, fmt.Errorf("invalid allocate response format")
	}

	status, _ := arr[0].(string)
	// Lua 脚本以单元素数组表示空队列与优先级老化重试
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

	// 解析成功结果（第一个元素是 "ok"）
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

// Pause 暂停通话分配
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

// ResumeResult 恢复结果
type ResumeResult struct {
	Score float64
}

// Resume 恢复通话分配
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

// DequeueResult 出队结果
type DequeueResult struct {
	WaitTime int64
}

// Dequeue 通话出队
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

// --- 通话信息查询 ---

// GetCallInfo 获取通话详情
func (r *AcdRedisRepo) GetCallInfo(ctx context.Context, callID string) (*model.CallInfo, error) {
	data, err := r.client.HGetAll(ctx, r.callKey(callID)).Result()
	if err != nil {
		return nil, fmt.Errorf("get call info failed: %w", err)
	}
	if len(data) == 0 {
		return nil, nil // 不存在
	}

	return hashToCallInfo(data), nil
}

// hashToCallInfo 将 Redis HASH 数据转为 CallInfo
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

// --- 队列长度查询 ---

// GetActiveQueueLen 获取活跃队列长度
func (r *AcdRedisRepo) GetActiveQueueLen(ctx context.Context, queueID int64) (int64, error) {
	return r.client.ZCard(ctx, r.activeKey(queueID)).Result()
}

// GetPausedQueueLen 获取暂停队列长度
func (r *AcdRedisRepo) GetPausedQueueLen(ctx context.Context, queueID int64) (int64, error) {
	return r.client.ZCard(ctx, r.pausedKey(queueID)).Result()
}

// GetWorkingQueueIDs 获取所有工作中的队列 ID
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

// --- 分布式锁 ---

// TryLock 尝试获取队列级分布式锁
func (r *AcdRedisRepo) TryLock(ctx context.Context, queueID int64, ttl time.Duration) (bool, error) {
	ok, err := r.client.SetNX(ctx, r.lockKey(queueID), "1", ttl).Result()
	if err != nil {
		return false, fmt.Errorf("try lock failed: %w", err)
	}
	return ok, nil
}

// Unlock 释放队列级分布式锁
func (r *AcdRedisRepo) Unlock(ctx context.Context, queueID int64) error {
	_, err := r.client.Del(ctx, r.lockKey(queueID)).Result()
	return err
}

// --- Pub/Sub ---

// Publish 发布事件
func (r *AcdRedisRepo) Publish(ctx context.Context, channel string, message string) error {
	return r.client.Publish(ctx, channel, message).Err()
}

// Subscribe 订阅事件
func (r *AcdRedisRepo) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return r.client.Subscribe(ctx, channels...)
}
