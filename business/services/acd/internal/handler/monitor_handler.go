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

// MonitorHandler 监控 HTTP 处理器
type MonitorHandler struct {
	monitorService service.MonitorService
}

// NewMonitorHandler 创建监控处理器
func NewMonitorHandler(monitorService service.MonitorService) *MonitorHandler {
	return &MonitorHandler{
		monitorService: monitorService,
	}
}

// MonitorQueue 单队列监控
// @Summary 队列实时监控
// @Description 获取指定队列的实时统计数据
// @Tags Monitor
// @Produce json
// @Param queueId path int true "队列 ID"
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

// MonitorAllQueues 全部队列监控
// @Summary 全部队列实时监控
// @Description 获取所有工作中队列的实时统计数据
// @Tags Monitor
// @Produce json
// @Param tenantId query int false "租户 ID（可选）"
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
