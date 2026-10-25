package repository

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"vocara/services/acd/internal/model"
)

// StatsRepo 统计数据仓储
type StatsRepo struct {
	client redis.UniversalClient
}

// NewStatsRepo 创建统计仓储
func NewStatsRepo(client redis.UniversalClient) *StatsRepo {
	return &StatsRepo{client: client}
}

// statsKey 构造统计 key
func (r *StatsRepo) statsKey(queueID int64) string {
	return fmt.Sprintf(keyStats, keyPrefix, queueID)
}

// activeKey 构造 active 队列 key
func (r *StatsRepo) activeKey(queueID int64) string {
	return fmt.Sprintf(keyActive, keyPrefix, queueID)
}

// pausedKey 构造 paused 队列 key
func (r *StatsRepo) pausedKey(queueID int64) string {
	return fmt.Sprintf(keyPaused, keyPrefix, queueID)
}

// callKey 构造 call key
func (r *StatsRepo) callKey(callID string) string {
	return fmt.Sprintf(keyCall, keyPrefix, callID)
}

// GetQueueStats 获取队列实时统计
func (r *StatsRepo) GetQueueStats(ctx context.Context, queueID int64) (*model.QueueStats, error) {
	// 1. 从 stats hash 取统计数据
	statsData, err := r.client.HGetAll(ctx, r.statsKey(queueID)).Result()
	if err != nil {
		return nil, fmt.Errorf("get queue stats failed: %w", err)
	}

	// 2. 实时计算 waiting 和 paused 数量
	waiting, err := r.client.ZCard(ctx, r.activeKey(queueID)).Result()
	if err != nil {
		return nil, fmt.Errorf("get active queue len failed: %w", err)
	}

	paused, err := r.client.ZCard(ctx, r.pausedKey(queueID)).Result()
	if err != nil {
		return nil, fmt.Errorf("get paused queue len failed: %w", err)
	}

	// 3. 获取最长等待时间和最长等待通话
	maxWaitTime := int64(0)
	longestCallID := ""
	if waiting > 0 {
		// 取队首（score 最小 = 优先级最高且等得最久）
		// 但我们要最长等待时间的，所以应该取最低优先级里等得最久的...
		// 实际上 maxWaitTime 应该是等待时间最长的那个
		// 用 ZRANGE 取队首，再用 eqTime 计算
		members, err := r.client.ZRange(ctx, r.activeKey(queueID), 0, -1).Result()
		if err == nil && len(members) > 0 {
			// 遍历所有通话，找等待最久的
			now := time.Now().Unix()
			for _, callID := range members {
				eqTimeStr, err := r.client.HGet(ctx, r.callKey(callID), "eqTime").Result()
				if err != nil {
					continue
				}
				eqTime, err := strconv.ParseInt(eqTimeStr, 10, 64)
				if err != nil {
					continue
				}
				waitTime := now - eqTime
				if waitTime > maxWaitTime {
					maxWaitTime = waitTime
					longestCallID = callID
				}
			}
		}
	}

	stats := &model.QueueStats{
		QueueID:      queueID,
		WaitingCount: waiting,
		PausedCount:  paused,
		ServingCount: parseInt64(statsData["serving"]),
		TotalToday:   parseInt64(statsData["total_today"]),
		AnsweredToday: parseInt64(statsData["answered_today"]),
		AbandonedToday: parseInt64(statsData["abandoned_today"]),
		AvgWaitTime:  parseFloat(statsData["avg_wait_time"]),
		MaxWaitTime:  maxWaitTime,
		LongestCallID: longestCallID,
		SLA:          parseFloat(statsData["sla"]),
		UpdateTime:   time.Now(),
	}

	return stats, nil
}

// IncrTotalToday 增加今日总入队数
func (r *StatsRepo) IncrTotalToday(ctx context.Context, queueID int64) error {
	_, err := r.client.HIncrBy(ctx, r.statsKey(queueID), "total_today", 1).Result()
	return err
}

// DecrServing 减少通话中计数（通话结束时调用）
func (r *StatsRepo) DecrServing(ctx context.Context, queueID int64) error {
	_, err := r.client.HIncrBy(ctx, r.statsKey(queueID), "serving", -1).Result()
	return err
}

// ResetDailyStats 重置每日统计（定时任务调用）
func (r *StatsRepo) ResetDailyStats(ctx context.Context, queueID int64) error {
	pipe := r.client.Pipeline()
	key := r.statsKey(queueID)
	pipe.HSet(ctx, key, "total_today", 0)
	pipe.HSet(ctx, key, "answered_today", 0)
	pipe.HSet(ctx, key, "abandoned_today", 0)
	pipe.HSet(ctx, key, "timeout_today", 0)
	pipe.HSet(ctx, key, "transferred_today", 0)
	pipe.HSet(ctx, key, "avg_wait_time", 0)
	pipe.HSet(ctx, key, "sla", 0)
	pipe.HSet(ctx, key, "sla_answered", 0)
	_, err := pipe.Exec(ctx)
	return err
}

// --- 辅助函数 ---

func parseInt64(s string) int64 {
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return v
}

func parseFloat(s string) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}
