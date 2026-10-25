package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"vocara/admin/internal/model"
	"vocara/pkg/errors"
)

// upsertCustomerFromForm 客户信息表单提交：以 data.phone  upsert 客户主档
func (s *FormService) upsertCustomerFromForm(ctx context.Context, tenantID uint64, data map[string]interface{}, now time.Time) (uint64, error) {
	phone := strings.TrimSpace(asString(data["phone"]))
	if phone == "" {
		return 0, errors.NewBizError(errors.CodeParamError, "客户手机号不能为空")
	}
	name := asString(data["customer_name"])
	return s.upsertCustomer(ctx, tenantID, phone, name, data, now)
}

// upsertCustomer 客户建档或合并更新，返回客户 ID
func (s *FormService) upsertCustomer(ctx context.Context, tenantID uint64, phone, name string, data map[string]interface{}, now time.Time) (uint64, error) {
	dataJSON, _ := json.Marshal(data)
	existing, err := s.customerRepo.GetByPhone(ctx, tenantID, phone)
	if err != nil {
		return 0, err
	}
	if existing == nil {
		c := &model.Customer{
			TenantID:    tenantID,
			Phone:       phone,
			Name:        name,
			Data:        string(dataJSON),
			CallCount:   1,
			FirstSeenAt: now,
			LastSeenAt:  now,
		}
		if err := s.customerRepo.Create(ctx, c); err != nil {
			return 0, err
		}
		return c.ID, nil
	}
	existing.Name = name
	existing.Data = string(dataJSON)
	existing.CallCount++
	existing.LastSeenAt = now
	if err := s.customerRepo.Save(ctx, existing); err != nil {
		return 0, err
	}
	return existing.ID, nil
}

// ensureTicketCustomer 工单提交：按 data.phone 关联客户；不存在则按该号码建客户
func (s *FormService) ensureTicketCustomer(ctx context.Context, tenantID uint64, data map[string]interface{}, now time.Time) (uint64, error) {
	phone := strings.TrimSpace(asString(data["phone"]))
	if phone == "" {
		return 0, errors.NewBizError(errors.CodeParamError, "工单缺少客户电话")
	}
	existing, err := s.customerRepo.GetByPhone(ctx, tenantID, phone)
	if err != nil {
		return 0, err
	}
	if existing != nil {
		return existing.ID, nil
	}
	// 工单带出的新客户：建档（call_count=0，客户来电计数由客户信息表单维护）
	c := &model.Customer{
		TenantID:    tenantID,
		Phone:       phone,
		Name:        asString(data["customer_name"]),
		Data:        "{}",
		CallCount:   0,
		FirstSeenAt: now,
		LastSeenAt:  now,
	}
	if err := s.customerRepo.Create(ctx, c); err != nil {
		return 0, err
	}
	return c.ID, nil
}

// ListCustomers 客户主档分页
func (s *FormService) ListCustomers(ctx context.Context, tenantID uint64, keyword string, page, pageSize int) ([]*model.Customer, int64, error) {
	return s.customerRepo.Page(ctx, tenantID, keyword, page, pageSize)
}

// GetCustomerDetail 客户详情 + 该客户关联记录（工单等）
func (s *FormService) GetCustomerDetail(ctx context.Context, id uint64) (*model.Customer, []*model.FormRecord, error) {
	c, err := s.customerRepo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	recs, err := s.recordRepo.ListByCustomer(ctx, id)
	return c, recs, err
}

// asString 安全取字符串
func asString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
