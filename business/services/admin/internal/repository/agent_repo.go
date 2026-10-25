package repository

import (
	"context"
	"strings"

	"vocara/admin/internal/model"
	"vocara/pkg/errors"
	"vocara/pkg/logger"

	"go.uber.org/zap"
)

// AgentRepository 座席数据访问层
type AgentRepository struct {
	BaseRepository
}

// NewAgentRepository 创建座席仓储
func NewAgentRepository() *AgentRepository {
	return &AgentRepository{}
}

// AgentPageQuery 座席分页查询条件
type AgentPageQuery struct {
	Page        int    `json:"page" query:"page"`
	PageSize    int    `json:"page_size" query:"page_size"`
	AgentNumber string `json:"agent_number" query:"agent_number"`
	AgentName   string `json:"agent_name" query:"agent_name"`
	Role        string `json:"role" query:"role"`
	Status      *int   `json:"status" query:"status"`
	WorkStatus  *int   `json:"work_status" query:"work_status"`
	OrgID       uint64 `json:"org_id" query:"org_id"`
	Keyword     string `json:"keyword" query:"keyword"`
}

// PageQuery 分页查询座席
func (r *AgentRepository) PageQuery(ctx context.Context, tenantID uint64, q *AgentPageQuery) ([]*model.Agent, int64, error) {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx).Model(&model.Agent{}), tenantID)

	if q.AgentNumber != "" {
		tx = tx.Where("agent_number LIKE ?", "%"+q.AgentNumber+"%")
	}
	if q.AgentName != "" {
		tx = tx.Where("agent_name LIKE ?", "%"+q.AgentName+"%")
	}
	if q.Role != "" {
		tx = tx.Where("role = ?", q.Role)
	}
	if q.Status != nil {
		tx = tx.Where("status = ?", *q.Status)
	}
	if q.WorkStatus != nil {
		tx = tx.Where("work_status = ?", *q.WorkStatus)
	}
	if q.OrgID > 0 {
		tx = tx.Where("org_id = ?", q.OrgID)
	}
	if q.Keyword != "" {
		keyword := "%" + q.Keyword + "%"
		tx = tx.Where("(agent_number LIKE ? OR agent_name LIKE ? OR mobile LIKE ?)", keyword, keyword, keyword)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		logger.ErrorCtx(ctx, "count agents failed", zap.Error(err))
		return nil, 0, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}

	result, err := r.Paginate(ctx, tx, q.Page, q.PageSize, &[]*model.Agent{})
	if err != nil {
		return nil, 0, err
	}

	list, ok := result.List.(*[]*model.Agent)
	if !ok {
		return nil, 0, errors.ErrSystemError
	}

	return *list, total, nil
}

// GetByAgentNum 根据工号查询座席
func (r *AgentRepository) GetByAgentNum(ctx context.Context, tenantID uint64, agentNum string) (*model.Agent, error) {
	db := r.GetDB()
	var agent model.Agent
	tx := WithTenantScope(db.WithContext(ctx), tenantID)
	if err := tx.Where("agent_number = ?", agentNum).First(&agent).Error; err != nil {
		if err.Error() == "record not found" {
			return nil, errors.ErrAgentNotFound
		}
		logger.ErrorCtx(ctx, "get agent by num failed", zap.Error(err), zap.String("agent_number", agentNum))
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return &agent, nil
}

// GetByID 根据ID查询座席
func (r *AgentRepository) GetByID(ctx context.Context, tenantID, id uint64) (*model.Agent, error) {
	var agent model.Agent
	if err := r.FirstByID(ctx, tenantID, &agent, id); err != nil {
		return nil, err
	}
	return &agent, nil
}

// Create 创建座席
func (r *AgentRepository) Create(ctx context.Context, agent *model.Agent) error {
	return r.BaseRepository.Create(ctx, agent)
}

// Update 更新座席
func (r *AgentRepository) Update(ctx context.Context, agent *model.Agent) error {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx).Model(&model.Agent{}), agent.TenantID)
	result := tx.Where("id = ?", agent.ID).Updates(map[string]interface{}{
		"agent_name": agent.AgentName,
		"mobile":     agent.Mobile,
		"email":      agent.Email,
		"avatar":     agent.Avatar,
		"gender":     agent.Gender,
		"dn_number":  agent.DN,
		"role":       agent.Role,
		"status":     agent.Status,
		"org_id":     agent.OrgID,
		"remark":     agent.Remark,
	})
	if result.Error != nil {
		logger.ErrorCtx(ctx, "update agent failed", zap.Error(result.Error), zap.Uint64("id", agent.ID))
		return errors.WrapWithCode(result.Error, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// UpdateProfile 更新座席自助资料（白名单列：姓名、手机、邮箱、头像、性别）。
// 角色、状态、分机、组织机构等敏感列不经过此方法，杜绝自助越权。
func (r *AgentRepository) UpdateProfile(ctx context.Context, tenantID, id uint64, fields map[string]interface{}) error {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx).Model(&model.Agent{}), tenantID)
	result := tx.Where("id = ?", id).Updates(fields)
	if result.Error != nil {
		logger.ErrorCtx(ctx, "update agent profile failed", zap.Error(result.Error), zap.Uint64("id", id))
		return errors.WrapWithCode(result.Error, errors.CodeDBError, errors.MsgDBError)
	}
	if result.RowsAffected == 0 {
		return errors.ErrAgentNotFound
	}
	return nil
}

// UpdatePassword 更新密码
func (r *AgentRepository) UpdatePassword(ctx context.Context, tenantID, id uint64, password string) error {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx).Model(&model.Agent{}), tenantID)
	result := tx.Where("id = ?", id).Update("password", password)
	if result.Error != nil {
		logger.ErrorCtx(ctx, "update agent password failed", zap.Error(result.Error), zap.Uint64("id", id))
		return errors.WrapWithCode(result.Error, errors.CodeDBError, errors.MsgDBError)
	}
	if result.RowsAffected == 0 {
		return errors.ErrAgentNotFound
	}
	return nil
}

// UpdateStatus 更新座席状态
func (r *AgentRepository) UpdateStatus(ctx context.Context, tenantID, id uint64, status int) error {
	return r.BaseRepository.UpdateStatus(ctx, tenantID, &model.Agent{}, id, status)
}

// Delete 删除座席
func (r *AgentRepository) Delete(ctx context.Context, tenantID, id uint64) error {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx), tenantID)
	result := tx.Where("id = ?", id).Delete(&model.Agent{})
	if result.Error != nil {
		logger.ErrorCtx(ctx, "delete agent failed", zap.Error(result.Error), zap.Uint64("id", id))
		return errors.WrapWithCode(result.Error, errors.CodeDBError, errors.MsgDBError)
	}
	if result.RowsAffected == 0 {
		return errors.ErrAgentNotFound
	}
	return nil
}

// BatchDelete 批量删除座席
func (r *AgentRepository) BatchDelete(ctx context.Context, tenantID uint64, ids []uint64) error {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx), tenantID)
	result := tx.Where("id IN ?", ids).Delete(&model.Agent{})
	if result.Error != nil {
		logger.ErrorCtx(ctx, "batch delete agents failed", zap.Error(result.Error))
		return errors.WrapWithCode(result.Error, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// ExistsByAgentNum 检查工号是否存在
func (r *AgentRepository) ExistsByAgentNum(ctx context.Context, tenantID uint64, agentNum string, excludeID uint64) (bool, error) {
	return r.ExistsByField(ctx, tenantID, &model.Agent{}, "agent_number", agentNum, excludeID)
}

// CountStatus 统计各状态座席数
func (r *AgentRepository) CountStatus(ctx context.Context, tenantID uint64) (map[string]int64, error) {
	db := r.GetDB()
	type result struct {
		Status int
		Count  int64
	}
	var results []result
	tx := WithTenantScope(db.WithContext(ctx).Model(&model.Agent{}), tenantID)
	if err := tx.Select("status, COUNT(*) as count").Group("status").Scan(&results).Error; err != nil {
		logger.ErrorCtx(ctx, "count agent status failed", zap.Error(err))
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}

	m := make(map[string]int64)
	for _, r := range results {
		switch r.Status {
		case 1:
			m["enabled"] = r.Count
		case 0:
			m["disabled"] = r.Count
		}
	}
	return m, nil
}

// ListByIDs 根据ID列表查询座席
func (r *AgentRepository) ListByIDs(ctx context.Context, tenantID uint64, ids []uint64) ([]*model.Agent, error) {
	db := r.GetDB()
	var list []*model.Agent
	tx := WithTenantScope(db.WithContext(ctx), tenantID)
	if err := tx.Where("id IN ?", ids).Find(&list).Error; err != nil {
		logger.ErrorCtx(ctx, "list agents by ids failed", zap.Error(err))
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return list, nil
}

// ===== 座席技能组关系操作 =====

// AddAgentSkills 批量添加座席技能关系
func (r *AgentRepository) AddAgentSkills(ctx context.Context, skills []*model.AgentSkill) error {
	return r.BatchCreate(ctx, skills)
}

// RemoveAgentSkills 删除座席的所有技能关系
func (r *AgentRepository) RemoveAgentSkills(ctx context.Context, tenantID, agentID uint64) error {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx), tenantID)
	result := tx.Where("agent_id = ?", agentID).Delete(&model.AgentSkill{})
	if result.Error != nil {
		logger.ErrorCtx(ctx, "remove agent skills failed", zap.Error(result.Error), zap.Uint64("agent_id", agentID))
		return errors.WrapWithCode(result.Error, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// GetAgentSkills 获取座席的技能列表
func (r *AgentRepository) GetAgentSkills(ctx context.Context, tenantID, agentID uint64) ([]*model.AgentSkill, error) {
	db := r.GetDB()
	var list []*model.AgentSkill
	tx := WithTenantScope(db.WithContext(ctx), tenantID)
	if err := tx.Where("agent_id = ?", agentID).Find(&list).Error; err != nil {
		logger.ErrorCtx(ctx, "get agent skills failed", zap.Error(err), zap.Uint64("agent_id", agentID))
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return list, nil
}

// UpdateDNByID 更新座席绑定的分机号
func (r *AgentRepository) UpdateDNByID(ctx context.Context, tenantID, agentID uint64, dn string) error {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx).Model(&model.Agent{}), tenantID)
	result := tx.Where("id = ?", agentID).Update("dn_number", dn)
	if result.Error != nil {
		logger.ErrorCtx(ctx, "update agent dn failed", zap.Error(result.Error), zap.Uint64("agent_id", agentID))
		return errors.WrapWithCode(result.Error, errors.CodeDBError, errors.MsgDBError)
	}
	return nil
}

// ListExport 导出座席列表
func (r *AgentRepository) ListExport(ctx context.Context, tenantID uint64, q *AgentPageQuery) ([]*model.Agent, error) {
	db := r.GetDB()
	tx := WithTenantScope(db.WithContext(ctx).Model(&model.Agent{}), tenantID)

	if q.AgentNumber != "" {
		tx = tx.Where("agent_number LIKE ?", "%"+q.AgentNumber+"%")
	}
	if q.AgentName != "" {
		tx = tx.Where("agent_name LIKE ?", "%"+q.AgentName+"%")
	}
	if q.Role != "" {
		tx = tx.Where("role = ?", q.Role)
	}
	if q.Status != nil {
		tx = tx.Where("status = ?", *q.Status)
	}
	if q.OrgID > 0 {
		tx = tx.Where("org_id = ?", q.OrgID)
	}
	if q.Keyword != "" {
		keyword := "%" + q.Keyword + "%"
		tx = tx.Where("(agent_number LIKE ? OR agent_name LIKE ? OR mobile LIKE ?)", keyword, keyword, keyword)
	}

	var list []*model.Agent
	if err := tx.Order("id DESC").Find(&list).Error; err != nil {
		logger.ErrorCtx(ctx, "export agents failed", zap.Error(err))
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return list, nil
}

// CountByOrgID 统计机构内座席数
func (r *AgentRepository) CountByOrgID(ctx context.Context, tenantID, orgID uint64) (int64, error) {
	var count int64
	tx := WithTenantScope(r.GetDB().WithContext(ctx).Model(&model.Agent{}), tenantID)
	if err := tx.Where("org_id = ?", orgID).Count(&count).Error; err != nil {
		return 0, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	return count, nil
}

// 忽略未使用变量
var _ = strings.Builder{}
