package repository

import (
	"context"
	"fmt"

	"vocara/services/agentcenter/internal/model"

	"gorm.io/gorm"
)

// OnlineAgentRepo provides online agent data access.
type OnlineAgentRepo struct {
	db *gorm.DB
}

// NewOnlineAgentRepo creates an online agent repository.
func NewOnlineAgentRepo(db *gorm.DB) *OnlineAgentRepo {
	return &OnlineAgentRepo{db: db}
}

// GetByAgentID returns an online agent record by agent ID.
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

// Create creates an online agent record.
func (r *OnlineAgentRepo) Create(ctx context.Context, online *model.OnlineAgent) error {
	return r.db.WithContext(ctx).Create(online).Error
}

// Update updates an online agent record.
func (r *OnlineAgentRepo) Update(ctx context.Context, online *model.OnlineAgent) error {
	return r.db.WithContext(ctx).Save(online).Error
}

// UpdateStatus updates the agent status.
func (r *OnlineAgentRepo) UpdateStatus(ctx context.Context, tenantID, agentID string, status int) error {
	return r.db.WithContext(ctx).
		Model(&model.OnlineAgent{}).
		Where("tenant_id = ? AND agent_id = ?", tenantID, agentID).
		Updates(map[string]interface{}{
			"status": status,
		}).Error
}

// UpdateHeartbeat updates the heartbeat time.
func (r *OnlineAgentRepo) UpdateHeartbeat(ctx context.Context, tenantID, agentID string) error {
	return r.db.WithContext(ctx).
		Model(&model.OnlineAgent{}).
		Where("tenant_id = ? AND agent_id = ?", tenantID, agentID).
		Update("last_heartbeat", gorm.Expr("NOW()")).Error
}

// Delete deletes an online agent record.
func (r *OnlineAgentRepo) Delete(ctx context.Context, tenantID, agentID string) error {
	return r.db.WithContext(ctx).
		Where("tenant_id = ? AND agent_id = ?", tenantID, agentID).
		Delete(&model.OnlineAgent{}).Error
}

// ListByTenant returns the online agents for a tenant.
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

// CountByTenant counts the online agents for a tenant.
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

// Upsert inserts or updates an online agent record.
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
