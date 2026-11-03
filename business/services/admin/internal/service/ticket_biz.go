package service

import (
	"context"

	"vocara/admin/internal/model"
	"vocara/pkg/errors"
)

// ListTickets returns a paginated list of tickets, filtering by status and matching keyword against ticket titles.
func (s *FormService) ListTickets(ctx context.Context, tenantID uint64, status, keyword string, page, pageSize int) ([]*model.FormRecord, int64, error) {
	return s.recordRepo.PageTickets(ctx, tenantID, status, keyword, page, pageSize)
}

// UpdateTicketStatus transitions a ticket between the open and closed states.
func (s *FormService) UpdateTicketStatus(ctx context.Context, id uint64, status string) error {
	if status != model.TicketStatusOpen && status != model.TicketStatusClosed {
		return errors.NewBizError(errors.CodeParamError, "工单状态非法："+status)
	}
	return s.recordRepo.SetBizStatus(ctx, id, status)
}

// GetTicketRecord retrieves a ticket record; its details are rendered using its version.
func (s *FormService) GetTicketRecord(ctx context.Context, tenantID, id uint64) (*model.FormRecord, error) {
	return s.recordRepo.GetByID(ctx, tenantID, id)
}
