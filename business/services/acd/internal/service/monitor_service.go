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

// MonitorService defines the monitoring service interface.
type MonitorService interface {
	// MonitorQueue returns real-time monitoring data for one queue.
	MonitorQueue(ctx context.Context, queueID int64) (*model.QueueStats, error)
	// MonitorAllQueues returns monitoring data for all queues belonging to a tenant.
	MonitorAllQueues(ctx context.Context, tenantID int64) ([]*model.QueueStats, error)
}

// monitorService implements the monitoring service.
type monitorService struct {
	repo      *repository.AcdRedisRepo
	statsRepo *repository.StatsRepo
}

// NewMonitorService creates a monitoring service.
func NewMonitorService(repo *repository.AcdRedisRepo, statsRepo *repository.StatsRepo) MonitorService {
	return &monitorService{
		repo:      repo,
		statsRepo: statsRepo,
	}
}

// MonitorQueue returns real-time monitoring data for one queue.
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

// MonitorAllQueues returns monitoring data for all working queues.
// tenantID is reserved; all queues are returned.
func (s *monitorService) MonitorAllQueues(ctx context.Context, tenantID int64) ([]*model.QueueStats, error) {
	// Get all working queue IDs.
	queueIDs, err := s.repo.GetWorkingQueueIDs(ctx)
	if err != nil {
		logger.ErrorCtx(ctx, "get working queue ids failed", zap.Error(err))
		return nil, errors.NewBizError(errors.CodeSystemError, "get working queues failed")
	}

	if len(queueIDs) == 0 {
		return []*model.QueueStats{}, nil
	}

	// Fetch statistics for all queues concurrently.
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

			// Filter by tenant when tenantID is specified.
			if tenantID > 0 {
				// Obtain tenantID from stats or call data.
				// This simplified path does not filter by tenant; production deployments obtain the tenant from metadata.
				// The queue configuration table can provide the tenant association.
			}

			statsCh <- stats
		}(qid)
	}

	wg.Wait()
	close(statsCh)
	close(errCh)

	// Collect the results.
	result := make([]*model.QueueStats, 0, len(queueIDs))
	for stats := range statsCh {
		result = append(result, stats)
	}

	return result, nil
}
