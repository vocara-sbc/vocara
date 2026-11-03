package handler

import (
	"context"

	"vocara/admin/internal/repository"
	"vocara/admin/internal/service"
	"vocara/pkg/response"

	"github.com/cloudwego/hertz/pkg/app"
)

// QueueHandler handles queue requests.
type QueueHandler struct {
	queueSvc *service.QueueService
}

// NewQueueHandler creates a queue handler.
func NewQueueHandler(queueSvc *service.QueueService) *QueueHandler {
	return &QueueHandler{
		queueSvc: queueSvc,
	}
}

// Page returns a paginated list of queues.
func (h *QueueHandler) Page(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	q := &repository.QueuePageQuery{}
	if err := c.BindAndValidate(q); err != nil {
		BindError(c, err)
		return
	}
	list, total, err := h.queueSvc.Page(ctx, tenantID, q)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询队列失败"))
		return
	}
	PageSuccess(c, list, total, q.Page, q.PageSize)
}

// List returns all queues.
func (h *QueueHandler) List(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	list, err := h.queueSvc.List(ctx, tenantID)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询队列列表失败"))
		return
	}
	response.OK(c, list)
}

// Get returns queue details.
func (h *QueueHandler) Get(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	id := ParseUintParam(c, "id")
	if id == 0 {
		response.FailParamError(c, "ID不能为空")
		return
	}
	queue, err := h.queueSvc.GetByID(ctx, tenantID, id)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询队列失败"))
		return
	}
	response.OK(c, queue)
}

// Save creates a queue.
func (h *QueueHandler) Save(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	dto := &service.QueueSaveDTO{}
	if err := c.BindAndValidate(dto); err != nil {
		BindError(c, err)
		return
	}
	queue, err := h.queueSvc.Create(ctx, tenantID, dto)
	if err != nil {
		response.Fail(c, WrapErr(err, "创建队列失败"))
		return
	}
	response.OK(c, queue)
}

// Update updates a queue.
func (h *QueueHandler) Update(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	dto := &service.QueueUpdateDTO{}
	if err := c.BindAndValidate(dto); err != nil {
		BindError(c, err)
		return
	}
	if dto.ID == 0 {
		dto.ID = ParseUintParam(c, "id")
	}
	if dto.ID == 0 {
		response.FailParamError(c, "ID不能为空")
		return
	}
	if err := h.queueSvc.Update(ctx, tenantID, dto); err != nil {
		response.Fail(c, WrapErr(err, "更新队列失败"))
		return
	}
	response.OKNoContent(c)
}

// Delete deletes a queue.
func (h *QueueHandler) Delete(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	id := ParseUintParam(c, "id")
	if id == 0 {
		response.FailParamError(c, "ID不能为空")
		return
	}
	if err := h.queueSvc.Delete(ctx, tenantID, id); err != nil {
		response.Fail(c, WrapErr(err, "删除队列失败"))
		return
	}
	response.OKNoContent(c)
}

// Monitor returns real-time queue monitoring data.
func (h *QueueHandler) Monitor(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	list, err := h.queueSvc.ListRealtime(ctx, tenantID)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询队列监控失败"))
		return
	}
	response.OK(c, list)
}

// Agents returns the skill group information associated with a queue's agents.
func (h *QueueHandler) Agents(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	id := ParseUintParam(c, "id")
	if id == 0 {
		response.FailParamError(c, "ID不能为空")
		return
	}
	queue, err := h.queueSvc.GetByID(ctx, tenantID, id)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询队列失败"))
		return
	}
	// Return queue information; agent details are resolved through skill group associations.
	response.OK(c, map[string]interface{}{
		"queue_id":  queue.ID,
		"skill_ids": queue.SkillIDs,
	})
}
