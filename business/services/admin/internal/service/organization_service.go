package service

import (
	"context"

	"vocara/admin/internal/model"
	"vocara/admin/internal/repository"
	"vocara/pkg/errors"
)

// OrganizationService provides organization management services.
type OrganizationService struct {
	orgRepo   *repository.OrganizationRepository
	agentRepo *repository.AgentRepository
}

// NewOrganizationService creates an organization service.
func NewOrganizationService() *OrganizationService {
	return &OrganizationService{
		orgRepo:   repository.NewOrganizationRepository(),
		agentRepo: repository.NewAgentRepository(),
	}
}

// OrganizationSaveDTO is a request to create an organization.
type OrganizationSaveDTO struct {
	OrgName    string `json:"org_name"`
	OrgCode    string `json:"org_code"`
	ParentID   uint64 `json:"parent_id"`
	LeaderID   uint64 `json:"leader_id"`
	LeaderName string `json:"leader_name"`
	Sort       int    `json:"sort"`
	Status     int    `json:"status"`
	Remark     string `json:"remark"`
}

// OrganizationUpdateDTO is a request to update an organization.
type OrganizationUpdateDTO struct {
	ID         uint64 `json:"id"`
	OrgName    string `json:"org_name"`
	ParentID   uint64 `json:"parent_id"`
	LeaderID   uint64 `json:"leader_id"`
	LeaderName string `json:"leader_name"`
	Sort       int    `json:"sort"`
	Status     int    `json:"status"`
	Remark     string `json:"remark"`
}

// List returns all organizations for building a tree.
func (s *OrganizationService) List(ctx context.Context, tenantID uint64) ([]*model.Organization, error) {
	return s.orgRepo.ListAll(ctx, tenantID)
}

// GetByID returns organization details.
func (s *OrganizationService) GetByID(ctx context.Context, tenantID, id uint64) (*model.Organization, error) {
	return s.orgRepo.GetByID(ctx, tenantID, id)
}

// Create creates an organization and calculates its level and path.
func (s *OrganizationService) Create(ctx context.Context, tenantID uint64, dto *OrganizationSaveDTO) (*model.Organization, error) {
	if dto.OrgName == "" {
		return nil, errors.ErrParamMissing
	}
	if dto.OrgCode != "" {
		exists, err := s.orgRepo.ExistsByCode(ctx, tenantID, dto.OrgCode, 0)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, errors.ErrAlreadyExists.WithDetail("机构编码已存在")
		}
	}

	level := 1
	path := "0"
	if dto.ParentID > 0 {
		parent, err := s.orgRepo.GetByID(ctx, tenantID, dto.ParentID)
		if err != nil {
			return nil, err
		}
		level = parent.Level + 1
		path = parent.Path + "," + uint64ToStr(parent.ID)
	}

	org := &model.Organization{
		BaseModel:  model.BaseModel{TenantID: tenantID},
		OrgName:    dto.OrgName,
		OrgCode:    dto.OrgCode,
		ParentID:   dto.ParentID,
		Level:      level,
		Path:       path,
		LeaderID:   dto.LeaderID,
		LeaderName: dto.LeaderName,
		Sort:       dto.Sort,
		Status:     dto.Status,
		Remark:     dto.Remark,
	}
	if org.Status == 0 {
		org.Status = 1
	}
	if err := s.orgRepo.Create(ctx, org); err != nil {
		return nil, err
	}
	return org, nil
}

// Update updates an organization and recalculates its level and path when its parent changes.
func (s *OrganizationService) Update(ctx context.Context, tenantID uint64, dto *OrganizationUpdateDTO) error {
	if dto.ID == 0 {
		return errors.ErrParamMissing
	}
	org, err := s.orgRepo.GetByID(ctx, tenantID, dto.ID)
	if err != nil {
		return err
	}

	if dto.ParentID != org.ParentID {
		if dto.ParentID == org.ID {
			return errors.ErrParamInvalid.WithDetail("不能将自己设为上级")
		}
		level := 1
		path := "0"
		if dto.ParentID > 0 {
			parent, err := s.orgRepo.GetByID(ctx, tenantID, dto.ParentID)
			if err != nil {
				return err
			}
			level = parent.Level + 1
			path = parent.Path + "," + uint64ToStr(parent.ID)
		}
		org.Level = level
		org.Path = path
	}

	org.OrgName = dto.OrgName
	org.ParentID = dto.ParentID
	org.LeaderID = dto.LeaderID
	org.LeaderName = dto.LeaderName
	org.Sort = dto.Sort
	org.Status = dto.Status
	org.Remark = dto.Remark
	return s.orgRepo.Update(ctx, org)
}

// Delete deletes an organization unless it has child organizations or members.
func (s *OrganizationService) Delete(ctx context.Context, tenantID, id uint64) error {
	children, err := s.orgRepo.CountChildren(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if children > 0 {
		return errors.ErrForbidden.WithDetail("存在子机构，无法删除")
	}
	agents, err := s.agentRepo.CountByOrgID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if agents > 0 {
		return errors.ErrForbidden.WithDetail("机构内存在座席，无法删除")
	}
	return s.orgRepo.Delete(ctx, tenantID, id)
}
