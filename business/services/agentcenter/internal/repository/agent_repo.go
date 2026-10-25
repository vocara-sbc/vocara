package repository

import (
	"context"
	"fmt"
	"strconv"

	"vocara/services/agentcenter/internal/model"

	"gorm.io/gorm"
)

// AgentRepo 座席数据访问
type AgentRepo struct {
	db *gorm.DB
}

// NewAgentRepo 创建座席仓库
func NewAgentRepo(db *gorm.DB) *AgentRepo {
	return &AgentRepo{db: db}
}

// GetByAgentID 根据座席ID获取座席
func (r *AgentRepo) GetByAgentID(ctx context.Context, tenantID, agentID string) (*model.Agent, error) {
	var agent model.Agent
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND agent_number = ?", tenantID, agentID).
		First(&agent).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get agent failed: %w", err)
	}
	return &agent, nil
}

// GetByID 根据主键ID获取座席
func (r *AgentRepo) GetByID(ctx context.Context, id uint) (*model.Agent, error) {
	var agent model.Agent
	err := r.db.WithContext(ctx).First(&agent, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get agent by id failed: %w", err)
	}
	return &agent, nil
}

// Create 创建座席
func (r *AgentRepo) Create(ctx context.Context, agent *model.Agent) error {
	return r.db.WithContext(ctx).Create(agent).Error
}

// Update 更新座席
func (r *AgentRepo) Update(ctx context.Context, agent *model.Agent) error {
	return r.db.WithContext(ctx).Save(agent).Error
}

// UpdateLastLogin 更新最后登录时间
func (r *AgentRepo) UpdateLastLogin(ctx context.Context, tenantID, agentID string) error {
	return r.db.WithContext(ctx).
		Model(&model.Agent{}).
		Where("tenant_id = ? AND agent_number = ?", tenantID, agentID).
		Update("last_login_at", gorm.Expr("NOW()")).Error
}

// ListByTenant 分页获取租户座席列表
func (r *AgentRepo) ListByTenant(ctx context.Context, tenantID string, page, pageSize int) ([]*model.Agent, int64, error) {
	var agents []*model.Agent
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Agent{}).Where("tenant_id = ?", tenantID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count agents failed: %w", err)
	}

	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Order("id DESC").Find(&agents).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list agents failed: %w", err)
	}
	return agents, total, nil
}

// GetAgentSkills 获取座席的技能组ID列表（agentID 为座席工号，返回 skill_group_id 字符串列表）
func (r *AgentRepo) GetAgentSkills(ctx context.Context, tenantID, agentID string) ([]string, error) {
	var ids []uint64
	err := r.db.WithContext(ctx).
		Model(&model.AgentSkill{}).
		Where("tenant_id = ? AND agent_number = ?", parseUintOrZero(tenantID), agentID).
		Pluck("skill_group_id", &ids).Error
	if err != nil {
		return nil, fmt.Errorf("get agent skills failed: %w", err)
	}
	skills := make([]string, 0, len(ids))
	for _, id := range ids {
		skills = append(skills, strconv.FormatUint(id, 10))
	}
	return skills, nil
}

// CreateServiceRecord 创建服务记录
func (r *AgentRepo) CreateServiceRecord(ctx context.Context, record *model.AgentServiceRecord) error {
	return r.db.WithContext(ctx).Create(record).Error
}

// UpdateServiceRecord 更新服务记录
func (r *AgentRepo) UpdateServiceRecord(ctx context.Context, record *model.AgentServiceRecord) error {
	return r.db.WithContext(ctx).Save(record).Error
}

// GetLatestServiceRecord 获取最新的服务记录
func (r *AgentRepo) GetLatestServiceRecord(ctx context.Context, tenantID, agentID string) (*model.AgentServiceRecord, error) {
	var record model.AgentServiceRecord
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND agent_id = ?", tenantID, agentID).
		Order("id DESC").
		First(&record).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get latest service record failed: %w", err)
	}
	return &record, nil
}
