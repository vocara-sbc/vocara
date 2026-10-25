package repository

import (
	"context"
	"time"

	"vocara/admin/internal/model"
	"vocara/pkg/errors"
	"vocara/pkg/logger"

	"go.uber.org/zap"
)

// RecordRepository 通话记录数据访问层
type RecordRepository struct {
	BaseRepository
}

// NewRecordRepository 创建通话记录仓储
func NewRecordRepository() *RecordRepository {
	return &RecordRepository{}
}

// RecordPageQuery 通话记录分页查询条件
type RecordPageQuery struct {
	Page           int       `json:"page" query:"page"`
	PageSize       int       `json:"page_size" query:"page_size"`
	CallType       *int      `json:"call_type" query:"call_type"`
	CallerNumber   string    `json:"caller_number" query:"caller_number"`
	CalledNumber   string    `json:"called_number" query:"called_number"`
	DID            string    `json:"did" query:"did"`
	AgentID        uint64    `json:"agent_id" query:"agent_id"`
	AgentNumber    string    `json:"agent_number" query:"agent_number"`
	QueueID        uint64    `json:"queue_id" query:"queue_id"`
	SkillGroupID   uint64    `json:"skill_id" query:"skill_id"`
	CallStatus     *int      `json:"call_status" query:"call_status"`
	AnswerStatus   *int      `json:"answer_status" query:"answer_status"`
	StartTime      time.Time `json:"start_time" query:"start_time"`
	EndTime        time.Time `json:"end_time" query:"end_time"`
	Keyword        string    `json:"keyword" query:"keyword"`
}

// PageQuery 分页查询通话记录
func (r *RecordRepository) PageQuery(ctx context.Context, tenantID uint64, q *RecordPageQuery) ([]*model.CallRecord, int64, error) {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx).Model(&model.CallRecord{}), tenantID)

	if q.CallType != nil {
		tx = tx.Where("call_type = ?", *q.CallType)
	}
	if q.CallerNumber != "" {
		tx = tx.Where("caller LIKE ?", "%"+q.CallerNumber+"%")
	}
	if q.CalledNumber != "" {
		tx = tx.Where("called LIKE ?", "%"+q.CalledNumber+"%")
	}
	if q.DID != "" {
		tx = tx.Where("did LIKE ?", "%"+q.DID+"%")
	}
	if q.AgentID > 0 {
		tx = tx.Where("agent_id = ?", q.AgentID)
	}
	if q.AgentNumber != "" {
		tx = tx.Where("agent_number LIKE ?", "%"+q.AgentNumber+"%")
	}
	if q.QueueID > 0 {
		tx = tx.Where("queue_id = ?", q.QueueID)
	}
	if q.SkillGroupID > 0 {
		tx = tx.Where("skill_id = ?", q.SkillGroupID)
	}
	if q.CallStatus != nil {
		tx = tx.Where("call_status = ?", *q.CallStatus)
	}
	if q.AnswerStatus != nil {
		tx = tx.Where("answer_status = ?", *q.AnswerStatus)
	}
	if !q.StartTime.IsZero() {
		tx = tx.Where("start_time >= ?", q.StartTime)
	}
	if !q.EndTime.IsZero() {
		tx = tx.Where("start_time <= ?", q.EndTime)
	}
	if q.Keyword != "" {
		keyword := "%" + q.Keyword + "%"
		tx = tx.Where("(caller LIKE ? OR called LIKE ? OR agent_number LIKE ?)", keyword, keyword, keyword)
	}

	result, err := r.Paginate(ctx, tx.Order("start_time DESC"), q.Page, q.PageSize, &[]*model.CallRecord{})
	if err != nil {
		return nil, 0, err
	}

	list, ok := result.List.(*[]*model.CallRecord)
	if !ok {
		return nil, 0, errors.ErrSystemError
	}
	return *list, result.Total, nil
}

// GetByID 根据ID查询通话记录
func (r *RecordRepository) GetByID(ctx context.Context, tenantID, id uint64) (*model.CallRecord, error) {
	var record model.CallRecord
	if err := r.FirstByID(ctx, tenantID, &record, id); err != nil {
		return nil, err
	}
	return &record, nil
}

// GetByCallID 根据CallID查询
func (r *RecordRepository) GetByCallID(ctx context.Context, tenantID uint64, callID string) (*model.CallRecord, error) {
	db := r.GetDB()
	var record model.CallRecord
	tx := WithTenantScope(db.WithContext(ctx), tenantID)
	if err := tx.Where("call_id = ?", callID).First(&record).Error; err != nil {
		if err.Error() == "record not found" {
			return nil, errors.ErrCallNotFound
		}
		logger.ErrorCtx(ctx, "get record by call id failed", zap.Error(err), zap.String("call_id", callID))
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return &record, nil
}

// ListExport 导出通话记录
func (r *RecordRepository) ListExport(ctx context.Context, tenantID uint64, q *RecordPageQuery) ([]*model.CallRecord, error) {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx).Model(&model.CallRecord{}), tenantID)

	if q.CallType != nil {
		tx = tx.Where("call_type = ?", *q.CallType)
	}
	if q.CallerNumber != "" {
		tx = tx.Where("caller LIKE ?", "%"+q.CallerNumber+"%")
	}
	if q.CalledNumber != "" {
		tx = tx.Where("called LIKE ?", "%"+q.CalledNumber+"%")
	}
	if q.DID != "" {
		tx = tx.Where("did LIKE ?", "%"+q.DID+"%")
	}
	if q.AgentID > 0 {
		tx = tx.Where("agent_id = ?", q.AgentID)
	}
	if q.QueueID > 0 {
		tx = tx.Where("queue_id = ?", q.QueueID)
	}
	if q.CallStatus != nil {
		tx = tx.Where("call_status = ?", *q.CallStatus)
	}
	if !q.StartTime.IsZero() {
		tx = tx.Where("start_time >= ?", q.StartTime)
	}
	if !q.EndTime.IsZero() {
		tx = tx.Where("start_time <= ?", q.EndTime)
	}

	var list []*model.CallRecord
	if err := tx.Order("start_time DESC").Find(&list).Error; err != nil {
		logger.ErrorCtx(ctx, "export call records failed", zap.Error(err))
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return list, nil
}

// CountByDate 按日期统计通话记录
type DateCount struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

// CountInboundByDate 统计呼入量按日期
func (r *RecordRepository) CountInboundByDate(ctx context.Context, tenantID uint64, startDate, endDate time.Time) ([]*DateCount, error) {
	db := r.GetDB()
	var results []*DateCount
	tx := WithTenantScope(db.WithContext(ctx).Model(&model.CallRecord{}), tenantID)
	err := tx.Select("DATE(start_time) as date, COUNT(*) as count").
		Where("call_type = ?", 1).
		Where("start_time BETWEEN ? AND ?", startDate, endDate).
		Group("DATE(start_time)").
		Order("date ASC").
		Scan(&results).Error
	if err != nil {
		logger.ErrorCtx(ctx, "count inbound by date failed", zap.Error(err))
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return results, nil
}

// CountOutboundByDate 统计呼出量按日期
func (r *RecordRepository) CountOutboundByDate(ctx context.Context, tenantID uint64, startDate, endDate time.Time) ([]*DateCount, error) {
	db := r.GetDB()
	var results []*DateCount
	tx := WithTenantScope(db.WithContext(ctx).Model(&model.CallRecord{}), tenantID)
	err := tx.Select("DATE(start_time) as date, COUNT(*) as count").
		Where("call_type = ?", 2).
		Where("start_time BETWEEN ? AND ?", startDate, endDate).
		Group("DATE(start_time)").
		Order("date ASC").
		Scan(&results).Error
	if err != nil {
		logger.ErrorCtx(ctx, "count outbound by date failed", zap.Error(err))
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return results, nil
}

// GetTodayStats 获取今日通话统计
func (r *RecordRepository) GetTodayStats(ctx context.Context, tenantID uint64) (total, answered, missed int64, err error) {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx).Model(&model.CallRecord{}), tenantID)
	tx = tx.Where("DATE(start_time) = CURDATE()")

	// 总数
	if err = tx.Count(&total).Error; err != nil {
		logger.ErrorCtx(ctx, "count today total calls failed", zap.Error(err))
		return 0, 0, 0, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}

	// 已接听
	tx2 := WithTenantScope(db.WithContext(ctx).Model(&model.CallRecord{}), tenantID)
	if err = tx2.Where("DATE(start_time) = CURDATE() AND answer_status = 1").Count(&answered).Error; err != nil {
		logger.ErrorCtx(ctx, "count today answered calls failed", zap.Error(err))
		return 0, 0, 0, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}

	// 未接听
	tx3 := WithTenantScope(db.WithContext(ctx).Model(&model.CallRecord{}), tenantID)
	if err = tx3.Where("DATE(start_time) = CURDATE() AND answer_status = 0").Count(&missed).Error; err != nil {
		logger.ErrorCtx(ctx, "count today missed calls failed", zap.Error(err))
		return 0, 0, 0, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}

	return total, answered, missed, nil
}

// ===== 聚合统计相关 =====

// AgentCallStats 座席通话统计
type AgentCallStats struct {
	AgentID       uint64 `gorm:"column:agent_id"`
	TotalCalls    int64  `gorm:"column:total_calls"`
	InboundCalls  int64  `gorm:"column:inbound_calls"`
	OutboundCalls int64  `gorm:"column:outbound_calls"`
	AnsweredCalls int64  `gorm:"column:answered_calls"`
	MissedCalls   int64  `gorm:"column:missed_calls"`
	TotalDuration int64  `gorm:"column:total_duration"`
	TalkDuration  int64  `gorm:"column:talk_duration"`
	TransferCount int64  `gorm:"column:transfer_count"`
	HoldCount     int64  `gorm:"column:hold_count"`
	Satisfaction  int64  `gorm:"column:satisfaction"`
	SatisfactionCount int64 `gorm:"column:satisfaction_count"`
}

// GetAgentCallStats 按座席ID分组统计通话数据
func (r *RecordRepository) GetAgentCallStats(ctx context.Context, tenantID uint64, agentIDs []uint64, startDate, endDate time.Time) (map[uint64]*AgentCallStats, error) {
	if len(agentIDs) == 0 {
		return make(map[uint64]*AgentCallStats), nil
	}
	db := r.GetDB()
	var results []*AgentCallStats
	tx := WithTenantScope(db.WithContext(ctx).Model(&model.CallRecord{}), tenantID)
	err := tx.Select(
		"agent_id, "+
			"COUNT(*) as total_calls, "+
			"SUM(CASE WHEN call_type = 1 THEN 1 ELSE 0 END) as inbound_calls, "+
			"SUM(CASE WHEN call_type = 2 THEN 1 ELSE 0 END) as outbound_calls, "+
			"SUM(CASE WHEN answer_status = 1 THEN 1 ELSE 0 END) as answered_calls, "+
			"SUM(CASE WHEN answer_status = 0 THEN 1 ELSE 0 END) as missed_calls, "+
			"COALESCE(SUM(total_duration), 0) as total_duration, "+
			"COALESCE(SUM(talk_duration), 0) as talk_duration, "+
			"COALESCE(SUM(transfer_count), 0) as transfer_count, "+
			"COALESCE(SUM(hold_count), 0) as hold_count, "+
			"COALESCE(SUM(satisfaction), 0) as satisfaction, "+
			"SUM(CASE WHEN satisfaction > 0 THEN 1 ELSE 0 END) as satisfaction_count",
	).
		Where("start_time >= ? AND start_time <= ?", startDate, endDate).
		Where("agent_id IN ?", agentIDs).
		Group("agent_id").
		Scan(&results).Error
	if err != nil {
		logger.ErrorCtx(ctx, "get agent call stats failed", zap.Error(err), zap.Uint64("tenant_id", tenantID))
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}

	statsMap := make(map[uint64]*AgentCallStats, len(results))
	for _, s := range results {
		statsMap[s.AgentID] = s
	}
	return statsMap, nil
}

// QueueCallStats 队列通话统计
type QueueCallStats struct {
	QueueID        uint64 `gorm:"column:queue_id"`
	TotalCalls     int64  `gorm:"column:total_calls"`
	AnsweredCalls  int64  `gorm:"column:answered_calls"`
	AbandonedCalls int64  `gorm:"column:abandoned_calls"`
	TotalWaitTime  int64  `gorm:"column:total_wait_time"`
	MaxWaitTime    int64  `gorm:"column:max_wait_time"`
	TotalTalkTime  int64  `gorm:"column:total_talk_time"`
}

// GetQueueCallStats 按队列ID分组统计通话数据
func (r *RecordRepository) GetQueueCallStats(ctx context.Context, tenantID uint64, queueIDs []uint64, startDate, endDate time.Time) (map[uint64]*QueueCallStats, error) {
	if len(queueIDs) == 0 {
		return make(map[uint64]*QueueCallStats), nil
	}
	db := r.GetDB()
	var results []*QueueCallStats
	tx := WithTenantScope(db.WithContext(ctx).Model(&model.CallRecord{}), tenantID)
	err := tx.Select(
		"queue_id, "+
			"COUNT(*) as total_calls, "+
			"SUM(CASE WHEN answer_status = 1 THEN 1 ELSE 0 END) as answered_calls, "+
			"SUM(CASE WHEN answer_status = 0 THEN 1 ELSE 0 END) as abandoned_calls, "+
			"COALESCE(SUM(wait_duration), 0) as total_wait_time, "+
			"COALESCE(MAX(wait_duration), 0) as max_wait_time, "+
			"COALESCE(SUM(talk_duration), 0) as total_talk_time",
	).
		Where("start_time >= ? AND start_time <= ?", startDate, endDate).
		Where("queue_id IN ?", queueIDs).
		Group("queue_id").
		Scan(&results).Error
	if err != nil {
		logger.ErrorCtx(ctx, "get queue call stats failed", zap.Error(err), zap.Uint64("tenant_id", tenantID))
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}

	statsMap := make(map[uint64]*QueueCallStats, len(results))
	for _, s := range results {
		statsMap[s.QueueID] = s
	}
	return statsMap, nil
}

// LineCallStats 线路通话统计（按业务号码聚合，service层映射到线路）
type LineCallStats struct {
	LineID        uint64 // 由service层填充
	DID string `gorm:"column:did"`
	TotalCalls    int64  `gorm:"column:total_calls"`
	AnsweredCalls int64  `gorm:"column:answered_calls"`
	FailedCalls   int64  `gorm:"column:failed_calls"`
}

// GetLineCallStats 按业务号码分组统计通话数据
// 说明：CallRecord 中没有 line_id 字段，通过 did 与线路关联
func (r *RecordRepository) GetLineCallStats(ctx context.Context, tenantID uint64, businessNumbers []string, startDate, endDate time.Time) (map[string]*LineCallStats, error) {
	db := r.GetDB()
	var results []*LineCallStats
	tx := WithTenantScope(db.WithContext(ctx).Model(&model.CallRecord{}), tenantID)
	tx = tx.Select(
		"did, "+
			"COUNT(*) as total_calls, "+
			"SUM(CASE WHEN answer_status = 1 THEN 1 ELSE 0 END) as answered_calls, "+
			"SUM(CASE WHEN answer_status = 0 THEN 1 ELSE 0 END) as failed_calls",
	).
		Where("start_time >= ? AND start_time <= ?", startDate, endDate).
		Group("did")

	if len(businessNumbers) > 0 {
		tx = tx.Where("did IN ?", businessNumbers)
	}

	err := tx.Scan(&results).Error
	if err != nil {
		logger.ErrorCtx(ctx, "get line call stats failed", zap.Error(err), zap.Uint64("tenant_id", tenantID))
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}

	statsMap := make(map[string]*LineCallStats, len(results))
	for _, s := range results {
		statsMap[s.DID] = s
	}
	return statsMap, nil
}
