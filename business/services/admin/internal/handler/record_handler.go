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

// RecordHandler handles call record requests.
type RecordHandler struct {
	recordSvc      *service.RecordService
	licenseManager *license.Manager
}

// NewRecordHandler creates a call record handler.
func NewRecordHandler(recordSvc *service.RecordService, licenseManager *license.Manager) *RecordHandler {
	return &RecordHandler{
		recordSvc:      recordSvc,
		licenseManager: licenseManager,
	}
}

// InboundPage returns a paginated list of inbound call records.
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

// OutboundPage returns a paginated list of outbound call records.
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

// Detail returns call details.
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

// Export exports call records.
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
	// Share query conditions with paginated queries (call_type, keyword, time range, and others), and let the service render CSV.
	// Prefix the output with a UTF-8 BOM so Excel opens it without garbled characters.
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

// Today returns today's call statistics.
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

// Uncalled returns a paginated list of unanswered calls.
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

// Recording returns recording information.
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
