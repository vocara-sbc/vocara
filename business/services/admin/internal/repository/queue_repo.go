package repository

import (
	"context"

	"vocara/admin/internal/model"
	"vocara/pkg/errors"
	"vocara/pkg/logger"

	"go.uber.org/zap"
)

// QueueRepository 队列数据访问层
type QueueRepository struct {
	BaseRepository
}

// NewQueueRepository 创建队列仓储
func NewQueueRepository() *QueueRepository {
	return &QueueRepository{}
}

// QueuePageQuery 队列分页查询条件
type QueuePageQuery struct {
	Page      int    `json:"page" query:"page"`
	PageSize  int    `json:"page_size" query:"page_size"`
	QueueName string `json:"queue_name" query:"queue_name"`
	QueueCode string `json:"queue_code" query:"queue_code"`
	QueueType *int   `json:"queue_type" query:"queue_type"`
	Strategy  string `json:"strategy" query:"strategy"`
	Status    *int   `json:"status" query:"status"`
	Keyword   string `json:"keyword" query:"keyword"`
}

// PageQuery 分页查询
func (r *QueueRepository) PageQuery(ctx context.Context, tenantID uint64, q *QueuePageQuery) ([]*model.Queue, int64, error) {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx).Model(&model.Queue{}), tenantID)

	if q.QueueName != "" {
		tx = tx.Where("queue_name LIKE ?", "%"+q.QueueName+"%")
	}
	if q.QueueCode != "" {
		tx = tx.Where("queue_code LIKE ?", "%"+q.QueueCode+"%")
	}
	if q.QueueType != nil {
		tx = tx.Where("queue_type = ?", *q.QueueType)
	}
	if q.Strategy != "" {
		tx = tx.Where("strategy = ?", q.Strategy)
	}
	if q.Status != nil {
		tx = tx.Where("status = ?", *q.Status)
	}
	if q.Keyword != "" {
		keyword := "%" + q.Keyword + "%"
		tx = tx.Where("(queue_name LIKE ? OR queue_code LIKE ?)", keyword, keyword)
	}

	result, err := r.Paginate(ctx, tx.Order("id DESC"), q.Page, q.PageSize, &[]*model.Queue{})
	if err != nil {
		return nil, 0, err
	}

	list, ok := result.List.(*[]*model.Queue)
	if !ok {
		return nil, 0, errors.ErrSystemError
	}
	return *list, result.Total, nil
}

// ListAll 查询所有队列
func (r *QueueRepository) ListAll(ctx context.Context, tenantID uint64) ([]*model.Queue, error) {
	var list []*model.Queue
	if err := r.BaseRepository.ListAll(ctx, tenantID, &list, "id DESC"); err != nil {
		return nil, err
	}
	return list, nil
}

// GetByID 根据ID查询
func (r *QueueRepository) GetByID(ctx context.Context, tenantID, id uint64) (*model.Queue, error) {
	var queue model.Queue
	if err := r.FirstByID(ctx, tenantID, &queue, id); err != nil {
		return nil, err
	}
	return &queue, nil
}

// GetByCode 根据编码查询
func (r *QueueRepository) GetByCode(ctx context.Context, tenantID uint64, code string) (*model.Queue, error) {
	db := r.GetDB()
	var queue model.Queue
	tx := WithTenantScope(db.WithContext(ctx), tenantID)
	if err := tx.Where("queue_code = ?", code).First(&queue).Error; err != nil {
		if err.Error() == "record not found" {
			return nil, errors.ErrQueueNotFound
		}
		logger.ErrorCtx(ctx, "get queue by code failed", zap.Error(err), zap.String("code", code))
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return &queue, nil
}

// Create 创建
func (r *QueueRepository) Create(ctx context.Context, queue *model.Queue) error {
	return r.BaseRepository.Create(ctx, queue)
}

// Update 更新
func (r *QueueRepository) Update(ctx context.Context, queue *model.Queue) error {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx).Model(&model.Queue{}), queue.TenantID)
	result := tx.Where("id = ?", queue.ID).Updates(map[string]interface{}{
		"queue_name":    queue.QueueName,
		"queue_type":    queue.QueueType,
		"strategy":      queue.Strategy,
		"max_wait_time": queue.MaxWaitTime,
		"max_wait_num":  queue.MaxWaitNum,
		"overflow_type": queue.OverflowType,
		"overflow_value": queue.OverflowValue,
		"no_agent_type": queue.NoAgentType,
		"wrap_up_time":  queue.WrapUpTime,
		"skill_ids":     queue.SkillIDs,
		"status":        queue.Status,
		"sound_file":    queue.SoundFile,
		"remark":        queue.Remark,
	})
	if result.Error != nil {
		logger.ErrorCtx(ctx, "update queue failed", zap.Error(result.Error), zap.Uint64("id", queue.ID))
		return errors.WrapWithCode(result.Error, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// Delete 删除
func (r *QueueRepository) Delete(ctx context.Context, tenantID, id uint64) error {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx), tenantID)
	result := tx.Where("id = ?", id).Delete(&model.Queue{})
	if result.Error != nil {
		logger.ErrorCtx(ctx, "delete queue failed", zap.Error(result.Error), zap.Uint64("id", id))
		return errors.WrapWithCode(result.Error, errors.CodeDBError, errors.MsgDBError)
	}
	if result.RowsAffected == 0 {
		return errors.ErrQueueNotFound
	}
	return nil
}

// ExistsByCode 检查编码是否存在
func (r *QueueRepository) ExistsByCode(ctx context.Context, tenantID uint64, code string, excludeID uint64) (bool, error) {
	return r.ExistsByField(ctx, tenantID, &model.Queue{}, "queue_code", code, excludeID)
}

// UpdateStatus 更新状态
func (r *QueueRepository) UpdateStatus(ctx context.Context, tenantID, id uint64, status int) error {
	return r.BaseRepository.UpdateStatus(ctx, tenantID, &model.Queue{}, id, status)
}

// UpdateRealtimeStats 更新队列实时统计
func (r *QueueRepository) UpdateRealtimeStats(ctx context.Context, tenantID, id uint64, waitNum, agentReadyNum, agentBusyNum int) error {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx).Model(&model.Queue{}), tenantID)
	result := tx.Where("id = ?", id).Updates(map[string]interface{}{
		"wait_num":        waitNum,
		"agent_ready_num": agentReadyNum,
		"agent_busy_num":  agentBusyNum,
	})
	if result.Error != nil {
		logger.ErrorCtx(ctx, "update queue realtime stats failed", zap.Error(result.Error), zap.Uint64("id", id))
		return errors.WrapWithCode(result.Error, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// ListRealtime 实时监控队列列表
func (r *QueueRepository) ListRealtime(ctx context.Context, tenantID uint64) ([]*model.Queue, error) {
	db := r.GetDB()
	var list []*model.Queue
	tx := WithTenantScope(db.WithContext(ctx), tenantID)
	if err := tx.Where("status = 1").Order("id DESC").Find(&list).Error; err != nil {
		logger.ErrorCtx(ctx, "list queue realtime failed", zap.Error(err))
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return list, nil
}
