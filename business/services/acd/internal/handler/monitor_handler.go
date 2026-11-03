package handler

import (
	"context"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
	"go.uber.org/zap"

	"vocara/pkg/logger"
	"vocara/pkg/response"
	"vocara/services/acd/internal/service"
)

// MonitorHandler handles monitoring HTTP requests.
type MonitorHandler struct {
	monitorService service.MonitorService
}

// NewMonitorHandler creates a monitoring handler.
func NewMonitorHandler(monitorService service.MonitorService) *MonitorHandler {
	return &MonitorHandler{
		monitorService: monitorService,
	}
}

// MonitorQueue monitors a single queue.
// @Summary Real-time queue monitoring
// @Description Get real-time statistics for a specific queue
// @Tags Monitor
// @Produce json
// @Param queueId path int true "Queue ID"
// @Success 200 {object} response.Response{data=model.QueueStats}
// @Router /api/v1/acd/monitor/queue/{queueId} [get]
func (h *MonitorHandler) MonitorQueue(ctx context.Context, c *app.RequestContext) {
	queueIDStr := c.Param("queueId")
	queueID, err := strconv.ParseInt(queueIDStr, 10, 64)
	if err != nil || queueID <= 0 {
		response.FailParamError(c, "invalid queueId")
		return
	}

	stats, err := h.monitorService.MonitorQueue(ctx, queueID)
	if err != nil {
		logger.ErrorCtx(ctx, "monitor queue failed",
			zap.Int64("queueId", queueID),
			zap.Error(err),
		)
		response.Error(c, err)
		return
	}

	response.Success(c, stats)
}

// MonitorAllQueues monitors all queues.
// @Summary Real-time monitoring for all queues
// @Description Get real-time statistics for all working queues
// @Tags Monitor
// @Produce json
// @Param tenantId query int false "Tenant ID (optional)"
// @Success 200 {object} response.Response{data=[]model.QueueStats}
// @Router /api/v1/acd/monitor/queues [get]
func (h *MonitorHandler) MonitorAllQueues(ctx context.Context, c *app.RequestContext) {
	tenantIDStr := c.Query("tenantId")
	var tenantID int64
	if tenantIDStr != "" {
		var err error
		tenantID, err = strconv.ParseInt(tenantIDStr, 10, 64)
		if err != nil {
			response.FailParamError(c, "invalid tenantId")
			return
		}
	}

	statsList, err := h.monitorService.MonitorAllQueues(ctx, tenantID)
	if err != nil {
		logger.ErrorCtx(ctx, "monitor all queues failed", zap.Error(err))
		response.Error(c, err)
		return
	}

	response.Success(c, statsList)
}
