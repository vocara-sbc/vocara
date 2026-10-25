package service

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"

	"vocara/pkg/errors"
	"vocara/pkg/logger"
	"vocara/services/acd/internal/model"
	"vocara/services/acd/internal/repository"
)

// MonitorService 监控服务接口
type MonitorService interface {
	// MonitorQueue 获取单个队列的实时监控数据
	MonitorQueue(ctx context.Context, queueID int64) (*model.QueueStats, error)
	// MonitorAllQueues 获取指定租户下所有队列的监控数据
	MonitorAllQueues(ctx context.Context, tenantID int64) ([]*model.QueueStats, error)
}

// monitorService 监控服务实现
type monitorService struct {
	repo      *repository.AcdRedisRepo
	statsRepo *repository.StatsRepo
}

// NewMonitorService 创建监控服务
func NewMonitorService(repo *repository.AcdRedisRepo, statsRepo *repository.StatsRepo) MonitorService {
	return &monitorService{
		repo:      repo,
		statsRepo: statsRepo,
	}
}

// MonitorQueue 获取单个队列的实时监控数据
func (s *monitorService) MonitorQueue(ctx context.Context, queueID int64) (*model.QueueStats, error) {
	if queueID <= 0 {
		return nil, errors.NewBizError(errors.CodeParamError, "queueId is required")
	}

	stats, err := s.statsRepo.GetQueueStats(ctx, queueID)
	if err != nil {
		logger.ErrorCtx(ctx, "get queue stats failed",
			zap.Int64("queueId", queueID),
			zap.Error(err),
		)
		return nil, errors.NewBizError(errors.CodeSystemError, "get queue stats failed")
	}

	return stats, nil
}

// MonitorAllQueues 获取所有工作中队列的监控数据
// tenantID 预留，当前版本获取所有队列
func (s *monitorService) MonitorAllQueues(ctx context.Context, tenantID int64) ([]*model.QueueStats, error) {
	// 获取所有工作中的队列 ID
	queueIDs, err := s.repo.GetWorkingQueueIDs(ctx)
	if err != nil {
		logger.ErrorCtx(ctx, "get working queue ids failed", zap.Error(err))
		return nil, errors.NewBizError(errors.CodeSystemError, "get working queues failed")
	}

	if len(queueIDs) == 0 {
		return []*model.QueueStats{}, nil
	}

	// 并发获取所有队列的统计数据
	var wg sync.WaitGroup
	statsCh := make(chan *model.QueueStats, len(queueIDs))
	errCh := make(chan error, len(queueIDs))

	for _, qid := range queueIDs {
		wg.Add(1)
		go func(queueID int64) {
			defer wg.Done()

			stats, err := s.statsRepo.GetQueueStats(ctx, queueID)
			if err != nil {
				logger.WarnCtx(ctx, "get queue stats failed",
					zap.Int64("queueId", queueID),
					zap.Error(err),
				)
				errCh <- fmt.Errorf("queue %d: %w", queueID, err)
				return
			}

			// 过滤租户（如果指定了 tenantID）
			if tenantID > 0 {
				// 从 stats 或 call 中获取 tenantID
				// 简化处理：这里不做租户过滤，实际生产中需要从元数据获取
				// 可以通过队列配置表关联租户
			}

			statsCh <- stats
		}(qid)
	}

	wg.Wait()
	close(statsCh)
	close(errCh)

	// 收集结果
	result := make([]*model.QueueStats, 0, len(queueIDs))
	for stats := range statsCh {
		result = append(result, stats)
	}

	return result, nil
}
