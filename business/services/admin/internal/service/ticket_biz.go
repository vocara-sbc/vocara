package service

import (
	"context"

	"vocara/admin/internal/model"
	"vocara/pkg/errors"
)

// ListTickets 工单分页：status 过滤、keyword 匹配工单标题
func (s *FormService) ListTickets(ctx context.Context, tenantID uint64, status, keyword string, page, pageSize int) ([]*model.FormRecord, int64, error) {
	return s.recordRepo.PageTickets(ctx, tenantID, status, keyword, page, pageSize)
}

// UpdateTicketStatus 工单状态流转 open/closed
func (s *FormService) UpdateTicketStatus(ctx context.Context, id uint64, status string) error {
	if status != model.TicketStatusOpen && status != model.TicketStatusClosed {
		return errors.NewBizError(errors.CodeParamError, "工单状态非法："+status)
	}
	return s.recordRepo.SetBizStatus(ctx, id, status)
}

// GetTicketRecord 取工单记录（详情按其版本渲染）
func (s *FormService) GetTicketRecord(ctx context.Context, tenantID, id uint64) (*model.FormRecord, error) {
	return s.recordRepo.GetByID(ctx, tenantID, id)
}
