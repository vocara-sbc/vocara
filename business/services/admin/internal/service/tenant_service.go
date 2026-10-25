package service

import (
	"context"
	"time"

	"vocara/admin/internal/model"
	"vocara/admin/internal/repository"
	"vocara/pkg/errors"
	"vocara/pkg/license"
)

// TenantService 租户服务
type TenantService struct {
	systemRepo     *repository.SystemRepository
	licenseManager *license.Manager
}

// NewTenantService 创建租户服务
func NewTenantService(licenseManager *license.Manager) *TenantService {
	return &TenantService{
		systemRepo:     repository.NewSystemRepository(),
		licenseManager: licenseManager,
	}
}

// TenantSaveDTO 租户保存
type TenantSaveDTO struct {
	TenantName   string `json:"tenant_name"`
	TenantCode   string `json:"tenant_code"`
	ContactName  string `json:"contact_name"`
	ContactPhone string `json:"contact_phone"`
	ContactEmail string `json:"contact_email"`
	Address      string `json:"address"`
	Status       int    `json:"status"`
	ExpireTime   string `json:"expire_time"`
	MaxAgents    int    `json:"max_agents"`
	MaxLines     int    `json:"max_lines"`
	Remark       string `json:"remark"`
}

// TenantUpdateDTO 租户更新
type TenantUpdateDTO struct {
	ID           uint64 `json:"id"`
	TenantName   string `json:"tenant_name"`
	ContactName  string `json:"contact_name"`
	ContactPhone string `json:"contact_phone"`
	ContactEmail string `json:"contact_email"`
	Address      string `json:"address"`
	Status       int    `json:"status"`
	ExpireTime   string `json:"expire_time"`
	MaxAgents    int    `json:"max_agents"`
	MaxLines     int    `json:"max_lines"`
	Remark       string `json:"remark"`
}

// Page 分页查询租户
func (s *TenantService) Page(ctx context.Context, q *repository.TenantPageQuery) ([]*model.Tenant, int64, error) {
	q.Page, q.PageSize = normalizePage(q.Page, q.PageSize)
	return s.systemRepo.PageQueryTenant(ctx, q)
}

// List 查询所有租户
func (s *TenantService) List(ctx context.Context) ([]*model.Tenant, error) {
	q := &repository.TenantPageQuery{
		Page:     1,
		PageSize: 1000,
	}
	list, _, err := s.systemRepo.PageQueryTenant(ctx, q)
	return list, err
}

// GetByID 根据ID查询租户
func (s *TenantService) GetByID(ctx context.Context, id uint64) (*model.Tenant, error) {
	return s.systemRepo.GetTenantByID(ctx, id)
}

// GetByCode 根据编码查询租户
func (s *TenantService) GetByCode(ctx context.Context, code string) (*model.Tenant, error) {
	return s.systemRepo.GetTenantByCode(ctx, code)
}

// Create 创建租户
func (s *TenantService) Create(ctx context.Context, dto *TenantSaveDTO) (*model.Tenant, error) {
	if dto.TenantName == "" || dto.TenantCode == "" {
		return nil, errors.ErrParamMissing
	}

	exists, err := s.systemRepo.ExistsTenantByCode(ctx, dto.TenantCode, 0)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.ErrAlreadyExists.WithDetail("租户编码已存在")
	}

	// 容量校验：租户数达授权上限则拒绝新增
	if s.licenseManager != nil {
		count, countErr := s.systemRepo.CountTenant(ctx)
		if countErr != nil {
			return nil, countErr
		}
		if !s.licenseManager.LimitAllowed(license.LimitTenants, int(count)) {
			return nil, errors.NewBizError(errors.CodeLicenseLimitExceeded, errors.MsgLicenseLimitExceeded).
				WithDetail("租户数已达授权上限")
		}
	}

	expireTime := time.Time{}
	if dto.ExpireTime != "" {
		expireTime, err = time.ParseInLocation("2006-01-02", dto.ExpireTime, time.Local)
		if err != nil {
			return nil, errors.ErrParamInvalid.WithDetail("过期时间格式错误")
		}
	}

	tenant := &model.Tenant{
		TenantName:   dto.TenantName,
		TenantCode:   dto.TenantCode,
		ContactName:  dto.ContactName,
		ContactPhone: dto.ContactPhone,
		ContactEmail: dto.ContactEmail,
		Address:      dto.Address,
		Status:       dto.Status,
		ExpireTime:   expireTime,
		MaxAgents:    dto.MaxAgents,
		MaxLines:     dto.MaxLines,
		Remark:       dto.Remark,
	}
	if tenant.Status == 0 {
		tenant.Status = 1
	}

	err = s.systemRepo.CreateTenant(ctx, tenant)
	if err != nil {
		return nil, err
	}

	return tenant, nil
}

// Update 更新租户
func (s *TenantService) Update(ctx context.Context, dto *TenantUpdateDTO) error {
	if dto.ID == 0 {
		return errors.ErrParamMissing
	}

	tenant, err := s.systemRepo.GetTenantByID(ctx, dto.ID)
	if err != nil {
		return err
	}

	if dto.ExpireTime != "" {
		expireTime, err := time.ParseInLocation("2006-01-02", dto.ExpireTime, time.Local)
		if err != nil {
			return errors.ErrParamInvalid.WithDetail("过期时间格式错误")
		}
		tenant.ExpireTime = expireTime
	}

	tenant.TenantName = dto.TenantName
	tenant.ContactName = dto.ContactName
	tenant.ContactPhone = dto.ContactPhone
	tenant.ContactEmail = dto.ContactEmail
	tenant.Address = dto.Address
	tenant.Status = dto.Status
	tenant.MaxAgents = dto.MaxAgents
	tenant.MaxLines = dto.MaxLines
	tenant.Remark = dto.Remark

	return s.systemRepo.UpdateTenant(ctx, tenant)
}

// Delete 删除租户
func (s *TenantService) Delete(ctx context.Context, id uint64) error {
	return s.systemRepo.DeleteTenant(ctx, id)
}

// UpdateStatus 更新租户状态
func (s *TenantService) UpdateStatus(ctx context.Context, id uint64, status int) error {
	return s.systemRepo.UpdateTenantStatus(ctx, id, status)
}
