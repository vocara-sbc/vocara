package service

import (
	"context"
	"strconv"

	"vocara/pkg/errors"
	"vocara/pkg/logger"
	"vocara/services/agentcenter/internal/model"
	"vocara/services/agentcenter/internal/repository"

	"go.uber.org/zap"
)

// SkillService 技能组服务
type SkillService struct {
	skillRepo  *repository.SkillRepo
	agentRepo  *repository.AgentRepo
	sessionSvc *SessionService
}

// NewSkillService 创建技能组服务
func NewSkillService(
	skillRepo *repository.SkillRepo,
	agentRepo *repository.AgentRepo,
	sessionSvc *SessionService,
) *SkillService {
	return &SkillService{
		skillRepo:  skillRepo,
		agentRepo:  agentRepo,
		sessionSvc: sessionSvc,
	}
}

// GetSkill 获取技能组详情
func (s *SkillService) GetSkill(ctx context.Context, tenantID, skillID string) (*model.SkillInfoResponse, error) {
	skill, err := s.skillRepo.GetBySkillID(ctx, tenantID, skillID)
	if err != nil {
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	if skill == nil {
		return nil, errors.ErrSkillNotFound
	}

	// 获取座席数量和空闲数量
	agents, err := s.skillRepo.GetAgentsBySkill(ctx, tenantID, skillID)
	if err != nil {
		logger.Warn("get agents by skill failed", zap.Error(err))
	}

	idleCount := int64(0)
	if count, err := s.sessionSvc.GetIdleCount(ctx, skillID); err == nil {
		idleCount = count
	}

	return &model.SkillInfoResponse{
		ID:          skill.ID,
		SkillCode:   skill.SkillCode,
		SkillName:   skill.SkillName,
		TenantID:    skill.TenantID,
		Description: skill.Description,
		Priority:    skill.Priority,
		Status:      skill.Status,
		AgentCount:  len(agents),
		IdleCount:   int(idleCount),
	}, nil
}

// ListSkill 技能组列表
func (s *SkillService) ListSkill(ctx context.Context, tenantID string, page, pageSize int) ([]*model.SkillInfoResponse, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	skills, total, err := s.skillRepo.ListByTenant(ctx, tenantID, page, pageSize)
	if err != nil {
		return nil, 0, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}

	result := make([]*model.SkillInfoResponse, 0, len(skills))
	for _, skill := range skills {
		agentCount := 0
		if agents, err := s.skillRepo.GetAgentsBySkill(ctx, tenantID, strconv.FormatUint(skill.ID, 10)); err == nil {
			agentCount = len(agents)
		}

		idleCount := 0
		if count, err := s.sessionSvc.GetIdleCount(ctx, strconv.FormatUint(skill.ID, 10)); err == nil {
			idleCount = int(count)
		}

		result = append(result, &model.SkillInfoResponse{
			ID:          skill.ID,
		SkillCode:   skill.SkillCode,
			SkillName:   skill.SkillName,
			TenantID:    skill.TenantID,
			Description: skill.Description,
			Priority:    skill.Priority,
			Status:      skill.Status,
			AgentCount:  agentCount,
			IdleCount:   idleCount,
		})
	}

	return result, total, nil
}

// GetSkillAgents 获取技能组下的座席列表
func (s *SkillService) GetSkillAgents(ctx context.Context, tenantID, skillID string) ([]*model.SkillAgentInfo, error) {
	relations, err := s.skillRepo.GetAgentsBySkill(ctx, tenantID, skillID)
	if err != nil {
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}

	result := make([]*model.SkillAgentInfo, 0, len(relations))
	for _, rel := range relations {
		agent, err := s.agentRepo.GetByAgentID(ctx, tenantID, rel.AgentNumber)
		if err != nil {
			logger.Warn("get agent failed", zap.Error(err), zap.String("agent_id", rel.AgentNumber))
			continue
		}
		if agent == nil {
			continue
		}

		// 获取实时状态
		status := model.StatusOffline
		session, err := s.sessionSvc.GetSession(ctx, tenantID, rel.AgentNumber)
		if err == nil && session != nil {
			status = session.GetStatus()
		}

		result = append(result, &model.SkillAgentInfo{
			AgentNumber: agent.AgentNumber,
			AgentName:  agent.AgentName,
			Status:     int(status),
			StatusName: status.String(),
			Level:      rel.SkillLevel,
		})
	}

	return result, nil
}
