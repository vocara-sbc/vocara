package service

import (
	"context"
	"time"

	"vocara/admin/internal/model"
	"vocara/admin/internal/repository"
	"vocara/pkg/logger"

	"go.uber.org/zap"
)

// StatsService provides statistics and report operations.
type StatsService struct {
	recordRepo *repository.RecordRepository
	agentRepo  *repository.AgentRepository
	queueRepo  *repository.QueueRepository
	lineRepo   *repository.LineRepository
}

// NewStatsService creates a statistics and report service.
func NewStatsService() *StatsService {
	return &StatsService{
		recordRepo: repository.NewRecordRepository(),
		agentRepo:  repository.NewAgentRepository(),
		queueRepo:  repository.NewQueueRepository(),
		lineRepo:   repository.NewLineRepository(),
	}
}

// StatsQuery contains statistics query filters.
type StatsQuery struct {
	StartDate string `json:"start_date" form:"start_date"`
	EndDate   string `json:"end_date" form:"end_date"`
	OrgID     uint64 `json:"org_id" form:"org_id"`
	AgentID   uint64 `json:"agent_id" form:"agent_id"`
	QueueID   uint64 `json:"queue_id" form:"queue_id"`
	LineID    uint64 `json:"line_id" form:"line_id"`
}

// GetAgentStats returns agent call traffic statistics.
func (s *StatsService) GetAgentStats(ctx context.Context, tenantID uint64, q *StatsQuery) ([]*model.AgentStats, error) {
	startDate, endDate := resolveDateRange(q.StartDate, q.EndDate)

	// Query the agent list.
	agentQ := &repository.AgentPageQuery{
		Page:     1,
		PageSize: 1000,
		OrgID:    q.OrgID,
	}
	if q.AgentID > 0 {
		agentQ.Page = 1
		agentQ.PageSize = 1
	}
	agents, _, err := s.agentRepo.PageQuery(ctx, tenantID, agentQ)
	if err != nil {
		logger.ErrorCtx(ctx, "get agents for stats failed", zap.Error(err))
		return nil, err
	}

	// Collect agent IDs for the SQL aggregate query.
	agentIDs := make([]uint64, 0, len(agents))
	result := make([]*model.AgentStats, 0, len(agents))
	for _, agent := range agents {
		agentIDs = append(agentIDs, agent.ID)
		stats := &model.AgentStats{
			TenantID:  tenantID,
			AgentID:   agent.ID,
			AgentNumber:  agent.AgentNumber,
			AgentName: agent.AgentName,
			OrgID:     agent.OrgID,
			StatDate:  startDate,
		}
		result = append(result, stats)
	}

	// Fetch statistics for all agents with one SQL GROUP BY query.
	if len(agentIDs) > 0 {
		statsMap, err := s.recordRepo.GetAgentCallStats(ctx, tenantID, agentIDs, startDate, endDate)
		if err != nil {
			logger.ErrorCtx(ctx, "get agent call stats failed", zap.Error(err))
			return nil, err
		}

		// Merge the aggregate results into the agent list.
		for _, stats := range result {
			if as, ok := statsMap[stats.AgentID]; ok {
				stats.TotalCalls = int(as.TotalCalls)
				stats.InboundCalls = int(as.InboundCalls)
				stats.OutboundCalls = int(as.OutboundCalls)
				stats.AnsweredCalls = int(as.AnsweredCalls)
				stats.MissedCalls = int(as.MissedCalls)
				stats.TotalDuration = int(as.TotalDuration)
				stats.TalkDuration = int(as.TalkDuration)
				stats.TransferCount = int(as.TransferCount)
				stats.HoldCount = int(as.HoldCount)
				if as.SatisfactionCount > 0 {
					stats.Satisfaction = float64(as.Satisfaction) / float64(as.SatisfactionCount)
				}
			}
		}
	}

	// Calculate averages.
	for _, stats := range result {
		if stats.AnsweredCalls > 0 {
			stats.AvgDuration = stats.TalkDuration / stats.AnsweredCalls
		}
	}

	return result, nil
}

// GetQueueStats returns queue statistics.
func (s *StatsService) GetQueueStats(ctx context.Context, tenantID uint64, q *StatsQuery) ([]*model.QueueStats, error) {
	startDate, endDate := resolveDateRange(q.StartDate, q.EndDate)

	queues, err := s.queueRepo.ListAll(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// Filter queues and collect their IDs.
	queueIDs := make([]uint64, 0, len(queues))
	result := make([]*model.QueueStats, 0, len(queues))
	for _, queue := range queues {
		if q.QueueID > 0 && queue.ID != q.QueueID {
			continue
		}
		queueIDs = append(queueIDs, queue.ID)
		stats := &model.QueueStats{
			TenantID:  tenantID,
			QueueID:   queue.ID,
			QueueName: queue.QueueName,
			QueueCode: queue.QueueCode,
			StatDate:  startDate,
		}
		result = append(result, stats)
	}

	// Fetch statistics for all queues with one SQL GROUP BY query.
	if len(queueIDs) > 0 {
		statsMap, err := s.recordRepo.GetQueueCallStats(ctx, tenantID, queueIDs, startDate, endDate)
		if err != nil {
			logger.ErrorCtx(ctx, "get queue call stats failed", zap.Error(err))
			return nil, err
		}

		// Merge the aggregate results into the queue list.
		for _, stats := range result {
			if qs, ok := statsMap[stats.QueueID]; ok {
				stats.TotalCalls = int(qs.TotalCalls)
				stats.AnsweredCalls = int(qs.AnsweredCalls)
				stats.AbandonedCalls = int(qs.AbandonedCalls)
				stats.MaxWaitTime = int(qs.MaxWaitTime)
				// Calculate averages from the aggregate totals.
				if qs.AnsweredCalls > 0 {
					stats.AvgWaitTime = int(qs.TotalWaitTime / qs.AnsweredCalls)
					stats.AvgTalkTime = int(qs.TotalTalkTime / qs.AnsweredCalls)
				}
			}
		}
	}

	// Calculate rates.
	for _, stats := range result {
		if stats.TotalCalls > 0 {
			stats.AnswerRate = float64(stats.AnsweredCalls) / float64(stats.TotalCalls) * 100
			stats.AbandonRate = float64(stats.AbandonedCalls) / float64(stats.TotalCalls) * 100
		}
	}

	return result, nil
}

// GetLineStats returns line statistics.
func (s *StatsService) GetLineStats(ctx context.Context, tenantID uint64, q *StatsQuery) ([]*model.LineStats, error) {
	startDate, endDate := resolveDateRange(q.StartDate, q.EndDate)

	lines, err := s.lineRepo.ListAll(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// Filter lines and collect DIDs for association queries.
	businessNumbers := make([]string, 0, len(lines))
	result := make([]*model.LineStats, 0, len(lines))
	bnToLineIndex := make(map[string][]int) // did -> result indices

	for _, line := range lines {
		if q.LineID > 0 && line.ID != q.LineID {
			continue
		}
		stats := &model.LineStats{
			TenantID:     tenantID,
			LineID:       line.ID,
			LineName:     line.LineName,
			LineCode:     line.LineCode,
			StatDate:     startDate,
			PeakChannels: 0,
		}
		idx := len(result)
		result = append(result, stats)

		// Associate the line's CallerID with the call record DID.
		if line.CallerID != "" {
			businessNumbers = append(businessNumbers, line.CallerID)
			bnToLineIndex[line.CallerID] = append(bnToLineIndex[line.CallerID], idx)
		}
	}

	// Fetch statistics grouped by DID with SQL GROUP BY.
	if len(businessNumbers) > 0 {
		statsMap, err := s.recordRepo.GetLineCallStats(ctx, tenantID, businessNumbers, startDate, endDate)
		if err != nil {
			logger.ErrorCtx(ctx, "get line call stats failed", zap.Error(err))
			return nil, err
		}

		// Merge the aggregate results into the line list.
		for bn, ls := range statsMap {
			indices, ok := bnToLineIndex[bn]
			if !ok {
				continue
			}
			for _, idx := range indices {
				result[idx].TotalCalls += int(ls.TotalCalls)
				result[idx].AnsweredCalls += int(ls.AnsweredCalls)
				result[idx].FailedCalls += int(ls.FailedCalls)
			}
		}
	}

	// Calculate success rates.
	for _, stats := range result {
		if stats.TotalCalls > 0 {
			stats.SuccessRate = float64(stats.AnsweredCalls) / float64(stats.TotalCalls) * 100
		}
	}

	return result, nil
}

// GetTrendStats returns trend statistics.
func (s *StatsService) GetTrendStats(ctx context.Context, tenantID uint64, q *StatsQuery) (map[string]interface{}, error) {
	startDate, endDate := resolveDateRange(q.StartDate, q.EndDate)

	inboundData, err := s.recordRepo.CountInboundByDate(ctx, tenantID, startDate, endDate)
	if err != nil {
		logger.ErrorCtx(ctx, "get inbound trend failed", zap.Error(err))
		return nil, err
	}

	outboundData, err := s.recordRepo.CountOutboundByDate(ctx, tenantID, startDate, endDate)
	if err != nil {
		logger.ErrorCtx(ctx, "get outbound trend failed", zap.Error(err))
		return nil, err
	}

	// Build date mappings.
	inboundMap := make(map[string]int64)
	for _, d := range inboundData {
		inboundMap[d.Date] = d.Count
	}
	outboundMap := make(map[string]int64)
	for _, d := range outboundData {
		outboundMap[d.Date] = d.Count
	}

	// Generate the complete date sequence.
	var dates []string
	var inboundCounts []int64
	var outboundCounts []int64

	start := startDate
	for !start.After(endDate) {
		dateStr := start.Format("2006-01-02")
		dates = append(dates, dateStr)
		inboundCounts = append(inboundCounts, inboundMap[dateStr])
		outboundCounts = append(outboundCounts, outboundMap[dateStr])
		start = start.AddDate(0, 0, 1)
	}

	return map[string]interface{}{
		"dates":      dates,
		"inbound":    inboundCounts,
		"outbound":   outboundCounts,
		"start_date": startDate.Format("2006-01-02"),
		"end_date":   endDate.Format("2006-01-02"),
	}, nil
}

// GetRealtimeStats returns real-time statistics.
func (s *StatsService) GetRealtimeStats(ctx context.Context, tenantID uint64) (*model.RealtimeStats, error) {
	stats := &model.RealtimeStats{}

	// Agent statistics.
	agentQ := &repository.AgentPageQuery{
		Page:     1,
		PageSize: 1000,
	}
	agents, _, err := s.agentRepo.PageQuery(ctx, tenantID, agentQ)
	if err != nil {
		return nil, err
	}
	stats.TotalAgents = len(agents)
	for _, agent := range agents {
		if agent.Status != 1 {
			continue
		}
		switch agent.WorkStatus {
		case 0:
			stats.OfflineAgents++
		case 1:
			stats.ReadyAgents++
			stats.OnlineAgents++
		case 2:
			stats.BusyAgents++
			stats.OnlineAgents++
		case 3:
			stats.BusyAgents++
			stats.OnlineAgents++
		case 4:
			stats.RestAgents++
			stats.OnlineAgents++
		case 5:
			stats.WrapUpAgents++
			stats.OnlineAgents++
		}
	}

	// Queue statistics.
	queues, err := s.queueRepo.ListAll(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	stats.TotalQueues = len(queues)
	for _, q := range queues {
		if q.Status == 1 {
			stats.ActiveQueues++
		}
		stats.WaitingCalls += q.WaitNum
	}

	// Line statistics.
	lines, err := s.lineRepo.ListAll(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	stats.TotalLines = len(lines)
	for _, line := range lines {
		if line.RegisterStatus == 1 {
			stats.OnlineLines++
		}
		stats.UsedChannels += line.UsedChannels
		stats.TotalChannels += line.MaxChannels
	}

	// Today's call statistics.
	total, answered, missed, err := s.recordRepo.GetTodayStats(ctx, tenantID)
	if err == nil {
		stats.TodayCalls = int(total)
		stats.TodayAnswered = int(answered)
		stats.TodayMissed = int(missed)
	}

	return stats, nil
}

// resolveDateRange parses a date range and defaults to the most recent seven days.
func resolveDateRange(startStr, endStr string) (time.Time, time.Time) {
	var startDate, endDate time.Time
	var err error

	if endStr != "" {
		endDate, err = time.ParseInLocation("2006-01-02", endStr, time.Local)
		if err != nil {
			endDate = time.Now()
		}
	} else {
		endDate = time.Now()
	}

	if startStr != "" {
		startDate, err = time.ParseInLocation("2006-01-02", startStr, time.Local)
		if err != nil {
			startDate = endDate.AddDate(0, 0, -6)
		}
	} else {
		startDate = endDate.AddDate(0, 0, -6)
	}

	// Normalize the range to day boundaries.
	startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())
	endDate = time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 0, endDate.Location())

	return startDate, endDate
}
