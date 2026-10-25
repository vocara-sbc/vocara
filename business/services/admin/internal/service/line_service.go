package service

import (
	"context"

	"vocara/admin/internal/model"
	"vocara/admin/internal/repository"
	"vocara/pkg/errors"
	"vocara/pkg/license"
)

// LineService 线路资源服务
type LineService struct {
	lineRepo       *repository.LineRepository
	licenseManager *license.Manager
}

// NewLineService 创建线路资源服务
func NewLineService(licenseManager *license.Manager) *LineService {
	return &LineService{
		lineRepo:       repository.NewLineRepository(),
		licenseManager: licenseManager,
	}
}

// LineSaveDTO 线路保存
type LineSaveDTO struct {
	LineName     string `json:"line_name"`
	LineCode     string `json:"line_code"`
	LineType     int    `json:"line_type"`
	Provider     string `json:"provider"`
	Gateway      string `json:"gateway"`
	Port         int    `json:"port"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	CallerID     string `json:"caller_id"`
	MaxChannels  int    `json:"max_channels"`
	Status       int    `json:"status"`
	SkillIDs     string `json:"skill_ids"`
	FSInstanceID uint64 `json:"fs_instance_id"`
	Remark       string `json:"remark"`
}

// LineUpdateDTO 线路更新
type LineUpdateDTO struct {
	ID           uint64 `json:"id"`
	LineName     string `json:"line_name"`
	LineType     int    `json:"line_type"`
	Provider     string `json:"provider"`
	Gateway      string `json:"gateway"`
	Port         int    `json:"port"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	CallerID     string `json:"caller_id"`
	MaxChannels  int    `json:"max_channels"`
	Status       int    `json:"status"`
	SkillIDs     string `json:"skill_ids"`
	FSInstanceID uint64 `json:"fs_instance_id"`
	Remark       string `json:"remark"`
}

// Page 分页查询
func (s *LineService) Page(ctx context.Context, tenantID uint64, q *repository.LinePageQuery) ([]*model.Line, int64, error) {
	q.Page, q.PageSize = normalizePage(q.Page, q.PageSize)
	return s.lineRepo.PageQuery(ctx, tenantID, q)
}

// List 查询列表
func (s *LineService) List(ctx context.Context, tenantID uint64) ([]*model.Line, error) {
	return s.lineRepo.ListAll(ctx, tenantID)
}

// GetByID 获取详情
func (s *LineService) GetByID(ctx context.Context, tenantID, id uint64) (*model.Line, error) {
	return s.lineRepo.GetByID(ctx, tenantID, id)
}

// Create 创建
func (s *LineService) Create(ctx context.Context, tenantID uint64, dto *LineSaveDTO) (*model.Line, error) {
	if dto.LineName == "" || dto.LineCode == "" {
		return nil, errors.ErrParamMissing
	}

	exists, err := s.lineRepo.ExistsByCode(ctx, tenantID, dto.LineCode, 0)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.ErrAlreadyExists.WithDetail("线路编码已存在")
	}

	// 容量校验：线路数达授权上限则拒绝新增
	if s.licenseManager != nil {
		count, countErr := s.lineRepo.Count(ctx, tenantID)
		if countErr != nil {
			return nil, countErr
		}
		if !s.licenseManager.LimitAllowed(license.LimitLines, int(count)) {
			return nil, errors.NewBizError(errors.CodeLicenseLimitExceeded, errors.MsgLicenseLimitExceeded).
				WithDetail("中继线路数已达授权上限")
		}
	}

	line := &model.Line{
		BaseModel: model.BaseModel{
			TenantID: tenantID,
		},
		LineName:     dto.LineName,
		LineCode:     dto.LineCode,
		LineType:     dto.LineType,
		Provider:     dto.Provider,
		Gateway:      dto.Gateway,
		Port:         dto.Port,
		Username:     dto.Username,
		Password:     dto.Password,
		CallerID:     dto.CallerID,
		MaxChannels:  dto.MaxChannels,
		Status:       dto.Status,
		SkillIDs:     dto.SkillIDs,
		FSInstanceID: dto.FSInstanceID,
		Remark:       dto.Remark,
	}
	if line.LineType == 0 {
		line.LineType = 1
	}
	if line.Port == 0 {
		line.Port = 5060
	}
	if line.MaxChannels == 0 {
		line.MaxChannels = 30
	}
	if line.Status == 0 {
		line.Status = 1
	}

	err = s.lineRepo.Create(ctx, line)
	if err != nil {
		return nil, err
	}

	return line, nil
}

// Update 更新
func (s *LineService) Update(ctx context.Context, tenantID uint64, dto *LineUpdateDTO) error {
	if dto.ID == 0 {
		return errors.ErrParamMissing
	}

	line, err := s.lineRepo.GetByID(ctx, tenantID, dto.ID)
	if err != nil {
		return err
	}

	line.LineName = dto.LineName
	line.LineType = dto.LineType
	line.Provider = dto.Provider
	line.Gateway = dto.Gateway
	line.Port = dto.Port
	line.Username = dto.Username
	if dto.Password != "" {
		line.Password = dto.Password
	}
	line.CallerID = dto.CallerID
	line.MaxChannels = dto.MaxChannels
	line.Status = dto.Status
	line.SkillIDs = dto.SkillIDs
	line.FSInstanceID = dto.FSInstanceID
	line.Remark = dto.Remark

	return s.lineRepo.Update(ctx, line)
}

// Delete 删除
func (s *LineService) Delete(ctx context.Context, tenantID, id uint64) error {
	return s.lineRepo.Delete(ctx, tenantID, id)
}

// UpdateStatus 更新状态
func (s *LineService) UpdateStatus(ctx context.Context, tenantID, id uint64, status int) error {
	return s.lineRepo.UpdateStatus(ctx, tenantID, id, status)
}

// GetStatus 获取线路状态统计
func (s *LineService) GetStatus(ctx context.Context, tenantID uint64) (map[string]interface{}, error) {
	total, used, err := s.lineRepo.SumChannels(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_channels": total,
		"used_channels":  used,
		"free_channels":  total - used,
	}, nil
}
