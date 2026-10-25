package handler

import (
	"context"

	"vocara/admin/internal/repository"
	"vocara/admin/internal/service"
	"vocara/pkg/response"

	"github.com/cloudwego/hertz/pkg/app"
)

// QueueHandler 队列处理器
type QueueHandler struct {
	queueSvc *service.QueueService
}

// NewQueueHandler 创建队列处理器
func NewQueueHandler(queueSvc *service.QueueService) *QueueHandler {
	return &QueueHandler{
		queueSvc: queueSvc,
	}
}

// Page 分页查询
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

// List 查询列表
func (h *QueueHandler) List(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	list, err := h.queueSvc.List(ctx, tenantID)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询队列列表失败"))
		return
	}
	response.OK(c, list)
}

// Get 获取详情
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

// Save 创建
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

// Update 更新
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

// Delete 删除
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

// Monitor 实时监控
func (h *QueueHandler) Monitor(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	list, err := h.queueSvc.ListRealtime(ctx, tenantID)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询队列监控失败"))
		return
	}
	response.OK(c, list)
}

// Agents 查询队列座席（简化：返回关联的技能组信息）
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
	// 返回队列信息，座席信息需要通过技能组关联
	response.OK(c, map[string]interface{}{
		"queue_id":  queue.ID,
		"skill_ids": queue.SkillIDs,
	})
}
