package service

import (
	"context"

	"vocara/pkg/errors"
	"vocara/pkg/logger"
	"vocara/services/agentcenter/internal/model"
	"vocara/services/agentcenter/internal/repository"

	"go.uber.org/zap"
)

// DnService 分机服务
type DnService struct {
	dnRepo     *repository.DnRepo
	sessionSvc *SessionService
}

// NewDnService 创建分机服务
func NewDnService(dnRepo *repository.DnRepo, sessionSvc *SessionService) *DnService {
	return &DnService{
		dnRepo:     dnRepo,
		sessionSvc: sessionSvc,
	}
}

// GetDn 获取分机详情
func (s *DnService) GetDn(ctx context.Context, tenantID, dnID string) (*model.DnInfoResponse, error) {
	dn, err := s.dnRepo.GetByDnID(ctx, tenantID, dnID)
	if err != nil {
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}
	if dn == nil {
		return nil, errors.ErrDNNotFound
	}

	return s.dnToResponse(ctx, dn), nil
}

// ListDn 分机列表
func (s *DnService) ListDn(ctx context.Context, tenantID string, page, pageSize int) ([]*model.DnInfoResponse, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	dns, total, err := s.dnRepo.ListByTenant(ctx, tenantID, page, pageSize)
	if err != nil {
		return nil, 0, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}

	result := make([]*model.DnInfoResponse, 0, len(dns))
	for _, dn := range dns {
		result = append(result, s.dnToResponse(ctx, dn))
	}

	return result, total, nil
}

// CheckOnline 批量检测分机在线
func (s *DnService) CheckOnline(ctx context.Context, tenantID string, dnNumbers []string) (map[string]bool, error) {
	result := make(map[string]bool, len(dnNumbers))
	for _, dn := range dnNumbers {
		result[dn] = false
	}

	// 从数据库获取分机信息
	dns, err := s.dnRepo.ListByNumbers(ctx, tenantID, dnNumbers)
	if err != nil {
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}

	// 检查座席是否在线（通过 session）
	for _, dn := range dns {
		if dn.AgentID != "" {
			online, err := s.sessionSvc.SessionExists(ctx, tenantID, dn.AgentID)
			if err != nil {
				logger.Warn("check session exists failed", zap.Error(err))
				continue
			}
			result[dn.DnNumber] = online
		}
	}

	return result, nil
}

// dnToResponse 转换为响应结构
func (s *DnService) dnToResponse(ctx context.Context, dn *model.Dn) *model.DnInfoResponse {
	// 默认为离线
	status := model.DnStatusOffline

	// 如果分机被禁用
	if dn.Status == 0 {
		status = model.DnStatusDisabled
	}

	online := false
	if dn.AgentID != "" && dn.Status == 1 {
		sessionExists, _ := s.sessionSvc.SessionExists(ctx, dn.TenantID, dn.AgentID)
		online = sessionExists

		// 如果座席在线，从 session 获取更精确的分机状态
		if sessionExists {
			session, err := s.sessionSvc.GetSession(ctx, dn.TenantID, dn.AgentID)
			if err == nil && session != nil {
				agentStatus := session.GetStatus()
				switch {
				case agentStatus == model.StatusRinging:
					status = model.DnStatusRinging
				case agentStatus.IsInCall():
					status = model.DnStatusBusy
				case agentStatus.IsIdle():
					status = model.DnStatusIdle
				case agentStatus == model.StatusBusy || agentStatus == model.StatusAfterWork:
					status = model.DnStatusBusy
				}
			}
		}
	}

	lastRegister := ""
	if dn.LastRegister != nil && !dn.LastRegister.IsZero() {
		lastRegister = dn.LastRegister.Format("2006-01-02 15:04:05")
	}

	return &model.DnInfoResponse{
		DnID:         dn.DnID,
		DnNumber:     dn.DnNumber,
		TenantID:     dn.TenantID,
		DisplayName:  dn.DisplayName,
		Status:       int(status),
		StatusName:   status.String(),
		AgentID:      dn.AgentID,
		RegisterIP:   dn.RegisterIP,
		LastRegister: lastRegister,
		Online:       online,
	}
}

// GetDnByNumber 根据分机号获取
func (s *DnService) GetDnByNumber(ctx context.Context, tenantID, dnNumber string) (*model.Dn, error) {
	dn, err := s.dnRepo.GetByDnNumber(ctx, tenantID, dnNumber)
	if err != nil {
		return nil, err
	}
	return dn, nil
}
