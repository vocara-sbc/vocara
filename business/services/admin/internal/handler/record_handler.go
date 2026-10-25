package handler

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"time"

	"vocara/admin/internal/repository"
	"vocara/admin/internal/service"
	"vocara/pkg/license"
	"vocara/pkg/response"

	"github.com/cloudwego/hertz/pkg/app"
)

// RecordHandler 通话记录处理器
type RecordHandler struct {
	recordSvc      *service.RecordService
	licenseManager *license.Manager
}

// NewRecordHandler 创建通话记录处理器
func NewRecordHandler(recordSvc *service.RecordService, licenseManager *license.Manager) *RecordHandler {
	return &RecordHandler{
		recordSvc:      recordSvc,
		licenseManager: licenseManager,
	}
}

// InboundPage 呼入记录分页
func (h *RecordHandler) InboundPage(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	q := &repository.RecordPageQuery{}
	if err := c.BindAndValidate(q); err != nil {
		BindError(c, err)
		return
	}
	list, total, err := h.recordSvc.InboundPage(ctx, tenantID, q)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询呼入记录失败"))
		return
	}
	PageSuccess(c, list, total, q.Page, q.PageSize)
}

// OutboundPage 呼出记录分页
func (h *RecordHandler) OutboundPage(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	q := &repository.RecordPageQuery{}
	if err := c.BindAndValidate(q); err != nil {
		BindError(c, err)
		return
	}
	list, total, err := h.recordSvc.OutboundPage(ctx, tenantID, q)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询呼出记录失败"))
		return
	}
	PageSuccess(c, list, total, q.Page, q.PageSize)
}

// Detail 通话详情
func (h *RecordHandler) Detail(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	id := ParseUintParam(c, "id")
	if id == 0 {
		response.FailParamError(c, "ID不能为空")
		return
	}
	record, err := h.recordSvc.GetDetail(ctx, tenantID, id)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询通话详情失败"))
		return
	}
	response.OK(c, record)
}

// Export 导出通话记录
func (h *RecordHandler) Export(ctx context.Context, c *app.RequestContext) {
	if err := CheckReportEnabled(h.licenseManager); err != nil {
		response.Fail(c, WrapErr(err, "报表导出失败"))
		return
	}
	tenantID := GetTenantID(ctx, c)
	q := &repository.RecordPageQuery{
		Page:     1,
		PageSize: 10000,
	}
	if err := c.BindAndValidate(q); err != nil {
		BindError(c, err)
		return
	}
	// 与分页查询共用查询条件（call_type/keyword/时间范围等），由 service 渲染 CSV；
	// 开头补 UTF-8 BOM，保证 Excel 直接打开不乱码
	var buf bytes.Buffer
	buf.Write([]byte{0xEF, 0xBB, 0xBF})
	if err := h.recordSvc.Export(ctx, tenantID, q, &buf); err != nil {
		response.Fail(c, WrapErr(err, "导出通话记录失败"))
		return
	}
	fileName := url.QueryEscape(fmt.Sprintf("通话记录_%s.csv", time.Now().Format("20060102150405")))
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", fileName))
	c.Data(200, "text/csv; charset=utf-8", buf.Bytes())
}

// Today 今日通话统计
func (h *RecordHandler) Today(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	total, answered, missed, err := h.recordSvc.GetTodayStats(ctx, tenantID)
	if err != nil {
		response.Fail(c, WrapErr(err, "获取今日通话统计失败"))
		return
	}
	response.OK(c, map[string]interface{}{
		"total":    total,
		"answered": answered,
		"missed":   missed,
	})
}

// Uncalled 未接通话列表
func (h *RecordHandler) Uncalled(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	page := ParseIntParam(c, "page")
	pageSize := ParseIntParam(c, "page_size")
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = 20
	}
	list, total, err := h.recordSvc.GetUncalled(ctx, tenantID, page, pageSize)
	if err != nil {
		response.Fail(c, WrapErr(err, "查询未接通话失败"))
		return
	}
	PageSuccess(c, list, total, page, pageSize)
}

// Recording 录音信息
func (h *RecordHandler) Recording(ctx context.Context, c *app.RequestContext) {
	tenantID := GetTenantID(ctx, c)
	id := ParseUintParam(c, "id")
	if id == 0 {
		response.FailParamError(c, "ID不能为空")
		return
	}
	url, err := h.recordSvc.GetRecordingURL(ctx, tenantID, id)
	if err != nil {
		response.Fail(c, WrapErr(err, "获取录音失败"))
		return
	}
	response.OK(c, map[string]interface{}{
		"record_id": id,
		"url":       url,
	})
}
