package service

import (
	"context"

	"vocara/pkg/errors"
	"vocara/pkg/logger"
	"vocara/services/agentcenter/internal/model"
	"vocara/services/agentcenter/internal/repository"

	"go.uber.org/zap"
)

// DnService provides extension operations.
type DnService struct {
	dnRepo     *repository.DnRepo
	sessionSvc *SessionService
}

// NewDnService creates an extension service.
func NewDnService(dnRepo *repository.DnRepo, sessionSvc *SessionService) *DnService {
	return &DnService{
		dnRepo:     dnRepo,
		sessionSvc: sessionSvc,
	}
}

// GetDn returns extension details.
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

// ListDn returns a paginated list of extensions.
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

// CheckOnline checks whether multiple extensions are online.
func (s *DnService) CheckOnline(ctx context.Context, tenantID string, dnNumbers []string) (map[string]bool, error) {
	result := make(map[string]bool, len(dnNumbers))
	for _, dn := range dnNumbers {
		result[dn] = false
	}

	// Load extension information from the database.
	dns, err := s.dnRepo.ListByNumbers(ctx, tenantID, dnNumbers)
	if err != nil {
		return nil, errors.WrapWithCode(err, errors.CodeDBError, errors.MsgDBError)
	}

	// Check whether each agent is online through its session.
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

// dnToResponse converts an extension to its response representation.
func (s *DnService) dnToResponse(ctx context.Context, dn *model.Dn) *model.DnInfoResponse {
	// Default to offline.
	status := model.DnStatusOffline

	// Handle a disabled extension.
	if dn.Status == 0 {
		status = model.DnStatusDisabled
	}

	online := false
	if dn.AgentID != "" && dn.Status == 1 {
		sessionExists, _ := s.sessionSvc.SessionExists(ctx, dn.TenantID, dn.AgentID)
		online = sessionExists

		// If the agent is online, obtain a more precise extension status from the session.
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

// GetDnByNumber returns an extension by number.
func (s *DnService) GetDnByNumber(ctx context.Context, tenantID, dnNumber string) (*model.Dn, error) {
	dn, err := s.dnRepo.GetByDnNumber(ctx, tenantID, dnNumber)
	if err != nil {
		return nil, err
	}
	return dn, nil
}
