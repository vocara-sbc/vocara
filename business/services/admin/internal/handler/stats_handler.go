package handler

import (
	"context"

	"vocara/admin/internal/service"
	"vocara/pkg/response"

	"github.com/cloudwego/hertz/pkg/app"
)

// StatsHandler handles statistics and report requests.
type StatsHandler struct {
	statsSvc *service.StatsService
}

// NewStatsHandler creates a statistics and report handler.
func NewStatsHandler(statsSvc *service.StatsService) *StatsHandler {
	return &StatsHandler{
		statsSvc: statsSvc,
	}
}

// AgentStats returns agent call traffic statistics.
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

// QueueStats returns queue statistics.
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

// LineStats returns line statistics.
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

// TrendStats returns trend statistics.
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

// RealtimeStats returns real-time statistics.
func (h *StatsHandler) RealtimeStats(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	stats, err := h.statsSvc.GetRealtimeStats(ctx, tenantID)
	if err != nil {
		response.Fail(c, WrapErr(err, "获取实时统计失败"))
		return
	}
	response.OK(c, stats)
}
