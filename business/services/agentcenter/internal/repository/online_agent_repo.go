package repository

import (
	"context"
	"fmt"

	"vocara/services/agentcenter/internal/model"

	"gorm.io/gorm"
)

// OnlineAgentRepo 在线座席数据访问
type OnlineAgentRepo struct {
	db *gorm.DB
}

// NewOnlineAgentRepo 创建在线座席仓库
func NewOnlineAgentRepo(db *gorm.DB) *OnlineAgentRepo {
	return &OnlineAgentRepo{db: db}
}

// GetByAgentID 根据座席ID获取在线记录
func (r *OnlineAgentRepo) GetByAgentID(ctx context.Context, tenantID, agentID string) (*model.OnlineAgent, error) {
	var online model.OnlineAgent
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND agent_id = ?", tenantID, agentID).
		First(&online).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get online agent failed: %w", err)
	}
	return &online, nil
}

// Create 创建在线座席记录
func (r *OnlineAgentRepo) Create(ctx context.Context, online *model.OnlineAgent) error {
	return r.db.WithContext(ctx).Create(online).Error
}

// Update 更新在线座席记录
func (r *OnlineAgentRepo) Update(ctx context.Context, online *model.OnlineAgent) error {
	return r.db.WithContext(ctx).Save(online).Error
}

// UpdateStatus 更新座席状态
func (r *OnlineAgentRepo) UpdateStatus(ctx context.Context, tenantID, agentID string, status int) error {
	return r.db.WithContext(ctx).
		Model(&model.OnlineAgent{}).
		Where("tenant_id = ? AND agent_id = ?", tenantID, agentID).
		Updates(map[string]interface{}{
			"status": status,
		}).Error
}

// UpdateHeartbeat 更新心跳时间
func (r *OnlineAgentRepo) UpdateHeartbeat(ctx context.Context, tenantID, agentID string) error {
	return r.db.WithContext(ctx).
		Model(&model.OnlineAgent{}).
		Where("tenant_id = ? AND agent_id = ?", tenantID, agentID).
		Update("last_heartbeat", gorm.Expr("NOW()")).Error
}

// Delete 删除在线座席记录
func (r *OnlineAgentRepo) Delete(ctx context.Context, tenantID, agentID string) error {
	return r.db.WithContext(ctx).
		Where("tenant_id = ? AND agent_id = ?", tenantID, agentID).
		Delete(&model.OnlineAgent{}).Error
}

// ListByTenant 获取租户在线座席列表
func (r *OnlineAgentRepo) ListByTenant(ctx context.Context, tenantID string, page, pageSize int) ([]*model.OnlineAgent, int64, error) {
	var list []*model.OnlineAgent
	var total int64

	query := r.db.WithContext(ctx).Model(&model.OnlineAgent{}).Where("tenant_id = ?", tenantID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count online agents failed: %w", err)
	}

	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Order("checkin_time DESC").Find(&list).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list online agents failed: %w", err)
	}
	return list, total, nil
}

// CountByTenant 统计租户在线座席数
func (r *OnlineAgentRepo) CountByTenant(ctx context.Context, tenantID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.OnlineAgent{}).
		Where("tenant_id = ?", tenantID).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("count online agents failed: %w", err)
	}
	return count, nil
}

// Upsert 插入或更新在线座席记录
func (r *OnlineAgentRepo) Upsert(ctx context.Context, online *model.OnlineAgent) error {
	existing, err := r.GetByAgentID(ctx, online.TenantID, online.AgentID)
	if err != nil {
		return err
	}
	if existing == nil {
		return r.Create(ctx, online)
	}
	online.ID = existing.ID
	return r.Update(ctx, online)
}
