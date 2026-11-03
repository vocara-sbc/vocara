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

// SkillService provides skill group operations.
type SkillService struct {
	skillRepo  *repository.SkillRepo
	agentRepo  *repository.AgentRepo
	sessionSvc *SessionService
}

// NewSkillService creates a skill group service.
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

// GetSkill returns skill group details.
func (s *SkillService) GetSkill(ctx context.Context, tenantID, skillID string) (*model.SkillInfoResponse, error) {
	skill, err := s.skillRepo.GetBySkillID(ctx, tenantID, skillID)
	if err != nil {
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	if skill == nil {
		return nil, errors.ErrSkillNotFound
	}

	// Get the agent count and idle count.
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

// ListSkill returns a paginated list of skill groups.
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

// GetSkillAgents returns the agents in a skill group.
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

		// Get the real-time status.
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
