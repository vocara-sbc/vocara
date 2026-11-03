package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"vocara/admin/internal/model"
	"vocara/pkg/errors"
)

// upsertCustomerFromForm upserts a customer master record using data.phone when a customer information form is submitted.
func (s *FormService) upsertCustomerFromForm(ctx context.Context, tenantID uint64, data map[string]interface{}, now time.Time) (uint64, error) {
	phone := strings.TrimSpace(asString(data["phone"]))
	if phone == "" {
		return 0, errors.NewBizError(errors.CodeParamError, "客户手机号不能为空")
	}
	name := asString(data["customer_name"])
	return s.upsertCustomer(ctx, tenantID, phone, name, data, now)
}

// upsertCustomer creates a customer record or merges updates into an existing one and returns the customer ID.
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

// ensureTicketCustomer associates a ticket submission with a customer using data.phone and creates a customer for that number if none exists.
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
	// Create a record for a new customer from a ticket with call_count=0; incoming call counts are maintained by the customer information form.
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

// ListCustomers returns a paginated list of customer master records.
func (s *FormService) ListCustomers(ctx context.Context, tenantID uint64, keyword string, page, pageSize int) ([]*model.Customer, int64, error) {
	return s.customerRepo.Page(ctx, tenantID, keyword, page, pageSize)
}

// GetCustomerDetail returns customer details and associated records such as tickets.
func (s *FormService) GetCustomerDetail(ctx context.Context, id uint64) (*model.Customer, []*model.FormRecord, error) {
	c, err := s.customerRepo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	recs, err := s.recordRepo.ListByCustomer(ctx, id)
	return c, recs, err
}

// asString safely extracts a string value.
func asString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
