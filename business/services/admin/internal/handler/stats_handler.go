package handler

import (
	"context"

	"vocara/admin/internal/service"
	"vocara/pkg/response"

	"github.com/cloudwego/hertz/pkg/app"
)

// StatsHandler 统计报表处理器
type StatsHandler struct {
	statsSvc *service.StatsService
}

// NewStatsHandler 创建统计报表处理器
func NewStatsHandler(statsSvc *service.StatsService) *StatsHandler {
	return &StatsHandler{
		statsSvc: statsSvc,
	}
}

// AgentStats 座席话务统计
func (h *StatsHandler) AgentStats(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	q := &service.StatsQuery{}
	if err := c.BindAndValidate(q); err != nil {
		BindError(c, err)
		return
	}
	stats, err := h.statsSvc.GetAgentStats(ctx, tenantID, q)
	if err != nil {
		response.Fail(c, WrapErr(err, "获取座席统计失败"))
		return
	}
	response.OK(c, stats)
}

// QueueStats 队列统计
func (h *StatsHandler) QueueStats(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	q := &service.StatsQuery{}
	if err := c.BindAndValidate(q); err != nil {
		BindError(c, err)
		return
	}
	stats, err := h.statsSvc.GetQueueStats(ctx, tenantID, q)
	if err != nil {
		response.Fail(c, WrapErr(err, "获取队列统计失败"))
		return
	}
	response.OK(c, stats)
}

// LineStats 线路统计
func (h *StatsHandler) LineStats(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	q := &service.StatsQuery{}
	if err := c.BindAndValidate(q); err != nil {
		BindError(c, err)
		return
	}
	stats, err := h.statsSvc.GetLineStats(ctx, tenantID, q)
	if err != nil {
		response.Fail(c, WrapErr(err, "获取线路统计失败"))
		return
	}
	response.OK(c, stats)
}

// TrendStats 趋势统计
func (h *StatsHandler) TrendStats(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	q := &service.StatsQuery{}
	if err := c.BindAndValidate(q); err != nil {
		BindError(c, err)
		return
	}
	stats, err := h.statsSvc.GetTrendStats(ctx, tenantID, q)
	if err != nil {
		response.Fail(c, WrapErr(err, "获取趋势统计失败"))
		return
	}
	response.OK(c, stats)
}

// RealtimeStats 实时统计
func (h *StatsHandler) RealtimeStats(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	stats, err := h.statsSvc.GetRealtimeStats(ctx, tenantID)
	if err != nil {
		response.Fail(c, WrapErr(err, "获取实时统计失败"))
		return
	}
	response.OK(c, stats)
}
