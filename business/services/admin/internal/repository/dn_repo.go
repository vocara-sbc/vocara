package repository

import (
	"context"

	"vocara/admin/internal/model"
	"vocara/pkg/errors"
	"vocara/pkg/logger"

	"go.uber.org/zap"
)

// DNRepository 分机数据访问层
type DNRepository struct {
	BaseRepository
}

// NewDNRepository 创建分机仓储
func NewDNRepository() *DNRepository {
	return &DNRepository{}
}

// DNPageQuery 分机分页查询条件
type DNPageQuery struct {
	Page           int    `json:"page" query:"page"`
	PageSize       int    `json:"page_size" query:"page_size"`
	DNNum          string `json:"dn_num" query:"dn_num"`
	DNName         string `json:"dn_name" query:"dn_name"`
	DNType         *int   `json:"dn_type" query:"dn_type"`
	Status         *int   `json:"status" query:"status"`
	RegisterStatus *int   `json:"register_status" query:"register_status"`
	AgentID        uint64 `json:"agent_id" query:"agent_id"`
	FSInstanceID   uint64 `json:"fs_instance_id" query:"fs_instance_id"`
	Keyword        string `json:"keyword" query:"keyword"`
}

// PageQuery 分页查询
func (r *DNRepository) PageQuery(ctx context.Context, tenantID uint64, q *DNPageQuery) ([]*model.DN, int64, error) {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx).Model(&model.DN{}), tenantID)

	if q.DNNum != "" {
		tx = tx.Where("dn_number LIKE ?", "%"+q.DNNum+"%")
	}
	if q.DNName != "" {
		tx = tx.Where("dn_name LIKE ?", "%"+q.DNName+"%")
	}
	if q.DNType != nil {
		tx = tx.Where("dn_type = ?", *q.DNType)
	}
	if q.Status != nil {
		tx = tx.Where("status = ?", *q.Status)
	}
	if q.RegisterStatus != nil {
		tx = tx.Where("register_status = ?", *q.RegisterStatus)
	}
	if q.AgentID > 0 {
		tx = tx.Where("agent_id = ?", q.AgentID)
	}
	if q.FSInstanceID > 0 {
		tx = tx.Where("fs_instance_id = ?", q.FSInstanceID)
	}
	if q.Keyword != "" {
		keyword := "%" + q.Keyword + "%"
		tx = tx.Where("(dn_num LIKE ? OR dn_name LIKE ?)", keyword, keyword)
	}

	result, err := r.Paginate(ctx, tx.Order("id DESC"), q.Page, q.PageSize, &[]*model.DN{})
	if err != nil {
		return nil, 0, err
	}

	list, ok := result.List.(*[]*model.DN)
	if !ok {
		return nil, 0, errors.ErrSystemError
	}
	return *list, result.Total, nil
}

// ListAll 查询所有分机
func (r *DNRepository) ListAll(ctx context.Context, tenantID uint64) ([]*model.DN, error) {
	var list []*model.DN
	if err := r.BaseRepository.ListAll(ctx, tenantID, &list, "id DESC"); err != nil {
		return nil, err
	}
	return list, nil
}

// GetByID 根据ID查询
func (r *DNRepository) GetByID(ctx context.Context, tenantID, id uint64) (*model.DN, error) {
	var dn model.DN
	if err := r.FirstByID(ctx, tenantID, &dn, id); err != nil {
		return nil, err
	}
	return &dn, nil
}

// GetByDNNum 根据分机号查询
func (r *DNRepository) GetByDNNum(ctx context.Context, tenantID uint64, dnNum string) (*model.DN, error) {
	db := r.GetDB()
	var dn model.DN
	tx := WithTenantScope(db.WithContext(ctx), tenantID)
	if err := tx.Where("dn_number = ?", dnNum).First(&dn).Error; err != nil {
		if err.Error() == "record not found" {
			return nil, errors.ErrDNNotFound
		}
		logger.ErrorCtx(ctx, "get dn by num failed", zap.Error(err), zap.String("dn_num", dnNum))
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return &dn, nil
}

// GetByAgentNum 按绑定座席工号查询分机（座席"我的分机"场景）
func (r *DNRepository) GetByAgentNum(ctx context.Context, tenantID uint64, agentNum string) (*model.DN, error) {
	db := r.GetDB()
	var dn model.DN
	tx := WithTenantScope(db.WithContext(ctx), tenantID)
	if err := tx.Where("agent_number = ?", agentNum).First(&dn).Error; err != nil {
		if err.Error() == "record not found" {
			return nil, errors.ErrDNNotFound
		}
		logger.ErrorCtx(ctx, "get dn by agent num failed", zap.Error(err), zap.String("agent_number", agentNum))
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return &dn, nil
}

// Create 创建
func (r *DNRepository) Create(ctx context.Context, dn *model.DN) error {
	return r.BaseRepository.Create(ctx, dn)
}

// Update 更新
func (r *DNRepository) Update(ctx context.Context, dn *model.DN) error {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx).Model(&model.DN{}), dn.TenantID)
	result := tx.Where("id = ?", dn.ID).Updates(map[string]interface{}{
		"dn_name":        dn.DNName,
		"password":       dn.Password,
		"dn_type":        dn.DNType,
		"agent_id":       dn.AgentID,
		"agent_number":      dn.AgentNumber,
		"status":         dn.Status,
		"fs_instance_id": dn.FSInstanceID,
		"remark":         dn.Remark,
	})
	if result.Error != nil {
		logger.ErrorCtx(ctx, "update dn failed", zap.Error(result.Error), zap.Uint64("id", dn.ID))
		return errors.WrapWithCode(result.Error, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// Delete 删除
func (r *DNRepository) Delete(ctx context.Context, tenantID, id uint64) error {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx), tenantID)
	result := tx.Where("id = ?", id).Delete(&model.DN{})
	if result.Error != nil {
		logger.ErrorCtx(ctx, "delete dn failed", zap.Error(result.Error), zap.Uint64("id", id))
		return errors.WrapWithCode(result.Error, errors.CodeDBError, errors.MsgDBError)
	}
	if result.RowsAffected == 0 {
		return errors.ErrDNNotFound
	}
	return nil
}

// ExistsByDNNum 检查分机号是否存在
func (r *DNRepository) ExistsByDNNum(ctx context.Context, tenantID uint64, dnNum string, excludeID uint64) (bool, error) {
	return r.ExistsByField(ctx, tenantID, &model.DN{}, "dn_number", dnNum, excludeID)
}

// UpdateStatus 更新状态
func (r *DNRepository) UpdateStatus(ctx context.Context, tenantID, id uint64, status int) error {
	return r.BaseRepository.UpdateStatus(ctx, tenantID, &model.DN{}, id, status)
}

// BindAgent 绑定座席
func (r *DNRepository) BindAgent(ctx context.Context, tenantID, id, agentID uint64, agentNum string) error {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx).Model(&model.DN{}), tenantID)
	result := tx.Where("id = ?", id).Updates(map[string]interface{}{
		"agent_id":  agentID,
		"agent_number": agentNum,
	})
	if result.Error != nil {
		logger.ErrorCtx(ctx, "bind agent to dn failed", zap.Error(result.Error), zap.Uint64("id", id))
		return errors.WrapWithCode(result.Error, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// UnbindAgent 解绑座席
func (r *DNRepository) UnbindAgent(ctx context.Context, tenantID, id uint64) error {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx).Model(&model.DN{}), tenantID)
	result := tx.Where("id = ?", id).Updates(map[string]interface{}{
		"agent_id":  0,
		"agent_number": "",
	})
	if result.Error != nil {
		logger.ErrorCtx(ctx, "unbind agent from dn failed", zap.Error(result.Error), zap.Uint64("id", id))
		return errors.WrapWithCode(result.Error, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// UpdateRegisterInfo 更新注册信息
func (r *DNRepository) UpdateRegisterInfo(ctx context.Context, tenantID, id uint64, registerStatus int, registerIP, registerUA string) error {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx).Model(&model.DN{}), tenantID)
	result := tx.Where("id = ?", id).Updates(map[string]interface{}{
		"register_status": registerStatus,
		"ip_address":      registerIP,
		"user_agent":      registerUA,
	})
	if result.Error != nil {
		logger.ErrorCtx(ctx, "update dn register info failed", zap.Error(result.Error), zap.Uint64("id", id))
		return errors.WrapWithCode(result.Error, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// BatchCreate 批量创建分机
func (r *DNRepository) BatchCreate(ctx context.Context, dns []*model.DN) error {
	return r.BaseRepository.BatchCreate(ctx, dns)
}

// ListAvailable 查询可用分机（未绑定座席）
func (r *DNRepository) ListAvailable(ctx context.Context, tenantID uint64) ([]*model.DN, error) {
	db := r.GetDB()
	var list []*model.DN
	tx := WithTenantScope(db.WithContext(ctx), tenantID)
	if err := tx.Where("agent_id = 0 AND status = 1").Order("dn_number ASC").Find(&list).Error; err != nil {
		logger.ErrorCtx(ctx, "list available dns failed", zap.Error(err))
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return list, nil
}
