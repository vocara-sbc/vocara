package service

import (
	"context"

	"vocara/admin/internal/model"
	"vocara/admin/internal/repository"
	"vocara/pkg/errors"
)

// OrganizationService 组织机构服务
type OrganizationService struct {
	orgRepo   *repository.OrganizationRepository
	agentRepo *repository.AgentRepository
}

// NewOrganizationService 创建组织机构服务
func NewOrganizationService() *OrganizationService {
	return &OrganizationService{
		orgRepo:   repository.NewOrganizationRepository(),
		agentRepo: repository.NewAgentRepository(),
	}
}

// OrganizationSaveDTO 创建机构请求
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

// OrganizationUpdateDTO 更新机构请求
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

// List 全部机构（树形用）
func (s *OrganizationService) List(ctx context.Context, tenantID uint64) ([]*model.Organization, error) {
	return s.orgRepo.ListAll(ctx, tenantID)
}

// GetByID 机构详情
func (s *OrganizationService) GetByID(ctx context.Context, tenantID, id uint64) (*model.Organization, error) {
	return s.orgRepo.GetByID(ctx, tenantID, id)
}

// Create 创建机构：计算层级与路径
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

// Update 更新机构：父级变化时重算层级路径
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

// Delete 删除机构：有子机构或成员时拒绝
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
