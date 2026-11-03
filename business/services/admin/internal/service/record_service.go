package service

import (
	"context"
	"encoding/csv"
	"io"
	"time"

	"vocara/admin/internal/model"
	"vocara/admin/internal/repository"
	"vocara/pkg/errors"
	"vocara/pkg/logger"

	"go.uber.org/zap"
)

// RecordService provides call record services.
type RecordService struct {
	recordRepo *repository.RecordRepository
}

// NewRecordService creates a call record service.
func NewRecordService() *RecordService {
	return &RecordService{
		recordRepo: repository.NewRecordRepository(),
	}
}

// InboundPage returns a paginated list of inbound call records.
func (s *RecordService) InboundPage(ctx context.Context, tenantID uint64, q *repository.RecordPageQuery) ([]*model.CallRecord, int64, error) {
	q.Page, q.PageSize = normalizePage(q.Page, q.PageSize)
	callType := 1
	q.CallType = &callType
	return s.recordRepo.PageQuery(ctx, tenantID, q)
}

// OutboundPage returns a paginated list of outbound call records.
func (s *RecordService) OutboundPage(ctx context.Context, tenantID uint64, q *repository.RecordPageQuery) ([]*model.CallRecord, int64, error) {
	q.Page, q.PageSize = normalizePage(q.Page, q.PageSize)
	callType := 2
	q.CallType = &callType
	return s.recordRepo.PageQuery(ctx, tenantID, q)
}

// GetDetail returns call record details.
func (s *RecordService) GetDetail(ctx context.Context, tenantID, id uint64) (*model.CallRecord, error) {
	return s.recordRepo.GetByID(ctx, tenantID, id)
}

// GetByCallID retrieves a call record by call ID.
func (s *RecordService) GetByCallID(ctx context.Context, tenantID uint64, callID string) (*model.CallRecord, error) {
	return s.recordRepo.GetByCallID(ctx, tenantID, callID)
}

// Export writes call records as CSV.
func (s *RecordService) Export(ctx context.Context, tenantID uint64, q *repository.RecordPageQuery, writer io.Writer) error {
	list, err := s.recordRepo.ListExport(ctx, tenantID, q)
	if err != nil {
		return err
	}

	csvWriter := csv.NewWriter(writer)
	defer csvWriter.Flush()

	// Write the header row.
	headers := []string{"通话ID", "通话类型", "主叫号码", "被叫号码", "业务号码",
		"座席工号", "座席姓名", "队列名称", "技能组",
		"开始时间", "接听时间", "结束时间", "总时长(秒)", "通话时长(秒)", "等待时长(秒)",
		"通话状态", "接听状态", "挂断原因", "挂断方",
		"录音文件", "满意度", "备注"}
	if err := csvWriter.Write(headers); err != nil {
		return err
	}

	for _, record := range list {
		callType := "呼入"
		if record.CallType == 2 {
			callType = "呼出"
		} else if record.CallType == 3 {
			callType = "内部呼叫"
		}
		callStatus := "未接"
		if record.CallStatus == 1 {
			callStatus = "已接"
		} else if record.CallStatus == 2 {
			callStatus = "已挂断"
		} else if record.CallStatus == 3 {
			callStatus = "转接"
		}
		answerStatus := "未接"
		if record.AnswerStatus == 1 {
			answerStatus = "已接"
		}
		hangupSide := "未知"
		if record.HangupSide == 1 {
			hangupSide = "主叫"
		} else if record.HangupSide == 2 {
			hangupSide = "被叫"
		} else if record.HangupSide == 3 {
			hangupSide = "系统"
		}
		row := []string{
			record.CallID,
			callType,
			record.CallerNumber,
			record.CalledNumber,
			record.DID,
			record.AgentNumber,
			record.AgentName,
			record.QueueName,
			record.SkillName,
			record.StartTime.Format(time.RFC3339),
			record.AnswerTime.Format(time.RFC3339),
			record.EndTime.Format(time.RFC3339),
			uint64ToStr(uint64(record.TotalDuration)),
			uint64ToStr(uint64(record.TalkDuration)),
			uint64ToStr(uint64(record.WaitDuration)),
			callStatus,
			answerStatus,
			record.HangupCause,
			hangupSide,
			record.RecordFile,
			uint64ToStr(uint64(record.Satisfaction)),
			record.Remark,
		}
		if err := csvWriter.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// GetTodayStats returns today's call statistics.
func (s *RecordService) GetTodayStats(ctx context.Context, tenantID uint64) (total, answered, missed int64, err error) {
	return s.recordRepo.GetTodayStats(ctx, tenantID)
}

// GetUncalled returns a paginated list of unanswered calls.
func (s *RecordService) GetUncalled(ctx context.Context, tenantID uint64, page, pageSize int) ([]*model.CallRecord, int64, error) {
	page, pageSize = normalizePage(page, pageSize)
	answerStatus := 0
	q := &repository.RecordPageQuery{
		Page:         page,
		PageSize:     pageSize,
		AnswerStatus: &answerStatus,
	}
	return s.recordRepo.PageQuery(ctx, tenantID, q)
}

// GetRecordingURL returns the recording URL.
func (s *RecordService) GetRecordingURL(ctx context.Context, tenantID, id uint64) (string, error) {
	record, err := s.recordRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return "", err
	}
	if record.RecordURL == "" && record.RecordFile == "" {
		return "", errors.ErrNotFound.WithDetail("录音文件不存在")
	}
	if record.RecordURL != "" {
		return record.RecordURL, nil
	}
	return record.RecordFile, nil
}

// GetTrendStats returns call trend statistics.
func (s *RecordService) GetTrendStats(ctx context.Context, tenantID uint64, startDate, endDate time.Time) (map[string]interface{}, error) {
	if startDate.IsZero() || endDate.IsZero() {
		// Default to the most recent 7 days.
		endDate = time.Now()
		startDate = endDate.AddDate(0, 0, -6)
	}

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

	// Build date-to-count maps.
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
		"dates":     dates,
		"inbound":   inboundCounts,
		"outbound":  outboundCounts,
		"start_date": startDate.Format("2006-01-02"),
		"end_date":   endDate.Format("2006-01-02"),
	}, nil
}
