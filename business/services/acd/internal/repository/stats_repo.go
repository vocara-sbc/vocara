package repository

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"vocara/services/acd/internal/model"
)

// StatsRepo provides statistics data access.
type StatsRepo struct {
	client redis.UniversalClient
}

// NewStatsRepo creates a statistics repository.
func NewStatsRepo(client redis.UniversalClient) *StatsRepo {
	return &StatsRepo{client: client}
}

// statsKey constructs the statistics key.
func (r *StatsRepo) statsKey(queueID int64) string {
	return fmt.Sprintf(keyStats, keyPrefix, queueID)
}

// activeKey constructs the active queue key.
func (r *StatsRepo) activeKey(queueID int64) string {
	return fmt.Sprintf(keyActive, keyPrefix, queueID)
}

// pausedKey constructs the paused queue key.
func (r *StatsRepo) pausedKey(queueID int64) string {
	return fmt.Sprintf(keyPaused, keyPrefix, queueID)
}

// callKey constructs the call key.
func (r *StatsRepo) callKey(callID string) string {
	return fmt.Sprintf(keyCall, keyPrefix, callID)
}

// GetQueueStats returns real-time queue statistics.
func (r *StatsRepo) GetQueueStats(ctx context.Context, queueID int64) (*model.QueueStats, error) {
	// 1. Read statistics from the stats hash.
	statsData, err := r.client.HGetAll(ctx, r.statsKey(queueID)).Result()
	if err != nil {
		return nil, fmt.Errorf("get queue stats failed: %w", err)
	}

	// 2. Calculate the waiting and paused counts in real time.
	waiting, err := r.client.ZCard(ctx, r.activeKey(queueID)).Result()
	if err != nil {
		return nil, fmt.Errorf("get active queue len failed: %w", err)
	}

	paused, err := r.client.ZCard(ctx, r.pausedKey(queueID)).Result()
	if err != nil {
		return nil, fmt.Errorf("get paused queue len failed: %w", err)
	}

	// 3. Find the longest wait time and the call with the longest wait.
	maxWaitTime := int64(0)
	longestCallID := ""
	if waiting > 0 {
		// The queue head has the lowest score, representing the highest priority and longest wait within that priority.
		// The maximum wait time may instead belong to the longest-waiting call at a lower priority.
		// maxWaitTime therefore requires the actual longest wait across the queue.
		// Use ZRANGE to inspect the queue and calculate wait times from eqTime.
		members, err := r.client.ZRange(ctx, r.activeKey(queueID), 0, -1).Result()
		if err == nil && len(members) > 0 {
			// Iterate over all calls to find the longest wait.
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

// IncrTotalToday increments today's total enqueue count.
func (r *StatsRepo) IncrTotalToday(ctx context.Context, queueID int64) error {
	_, err := r.client.HIncrBy(ctx, r.statsKey(queueID), "total_today", 1).Result()
	return err
}

// DecrServing decrements the in-progress call count when a call ends.
func (r *StatsRepo) DecrServing(ctx context.Context, queueID int64) error {
	_, err := r.client.HIncrBy(ctx, r.statsKey(queueID), "serving", -1).Result()
	return err
}

// ResetDailyStats resets daily statistics for a scheduled task.
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

// --- Helper functions ---

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
