package service

import (
	"context"

	"vocara/admin/internal/model"
	"vocara/admin/internal/repository"
	"vocara/pkg/errors"
)

// SystemService provides system management services.
type SystemService struct {
	systemRepo *repository.SystemRepository
}

// NewSystemService creates a system management service.
func NewSystemService() *SystemService {
	return &SystemService{
		systemRepo: repository.NewSystemRepository(),
	}
}

// ===== System configuration =====

// GetConfigList returns the system configuration list.
func (s *SystemService) GetConfigList(ctx context.Context, tenantID uint64) ([]*model.SystemConfig, error) {
	return s.systemRepo.ListSystemConfig(ctx, tenantID)
}

// GetConfig returns a system configuration entry.
func (s *SystemService) GetConfig(ctx context.Context, tenantID uint64, key string) (*model.SystemConfig, error) {
	return s.systemRepo.GetSystemConfig(ctx, tenantID, key)
}

// SaveConfig saves a system configuration entry.
func (s *SystemService) SaveConfig(ctx context.Context, config *model.SystemConfig) error {
	if config.ConfigKey == "" {
		return errors.ErrParamMissing
	}
	return s.systemRepo.SaveSystemConfig(ctx, config)
}

// DeleteConfig deletes a system configuration entry.
func (s *SystemService) DeleteConfig(ctx context.Context, id uint64) error {
	return s.systemRepo.DeleteSystemConfig(ctx, id)
}

// ===== FS instance management =====

// FSInstanceSaveDTO contains FS instance data for creation.
type FSInstanceSaveDTO struct {
	InstanceName string `json:"instance_name"`
	InstanceCode string `json:"instance_code"`
	Host         string `json:"host"`
	ESLPort      int    `json:"esl_port"`
	ESLPassword  string `json:"esl_password"`
	SIPPort      int    `json:"sip_port"`
	MaxSessions  int    `json:"max_sessions"`
	Status       int    `json:"status"`
	Remark       string `json:"remark"`
}

// FSInstanceUpdateDTO contains FS instance data for updates.
type FSInstanceUpdateDTO struct {
	ID           uint64 `json:"id"`
	InstanceName string `json:"instance_name"`
	Host         string `json:"host"`
	ESLPort      int    `json:"esl_port"`
	ESLPassword  string `json:"esl_password"`
	SIPPort      int    `json:"sip_port"`
	MaxSessions  int    `json:"max_sessions"`
	Status       int    `json:"status"`
	Remark       string `json:"remark"`
}

// FSPage returns a paginated list of FS instances.
func (s *SystemService) FSPage(ctx context.Context, q *repository.FSPageQuery) ([]*model.FSInstance, int64, error) {
	q.Page, q.PageSize = normalizePage(q.Page, q.PageSize)
	return s.systemRepo.PageQueryFS(ctx, q)
}

// FSList returns all FS instances.
func (s *SystemService) FSList(ctx context.Context) ([]*model.FSInstance, error) {
	return s.systemRepo.ListAllFS(ctx)
}

// FSGet returns FS instance details.
func (s *SystemService) FSGet(ctx context.Context, id uint64) (*model.FSInstance, error) {
	return s.systemRepo.GetFSByID(ctx, id)
}

// FSCreate creates an FS instance.
func (s *SystemService) FSCreate(ctx context.Context, dto *FSInstanceSaveDTO) (*model.FSInstance, error) {
	if dto.InstanceName == "" || dto.InstanceCode == "" || dto.Host == "" {
		return nil, errors.ErrParamMissing
	}

	exists, err := s.systemRepo.ExistsFSByCode(ctx, dto.InstanceCode, 0)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.ErrAlreadyExists.WithDetail("实例编码已存在")
	}

	fs := &model.FSInstance{
		InstanceName: dto.InstanceName,
		InstanceCode: dto.InstanceCode,
		Host:         dto.Host,
		ESLPort:      dto.ESLPort,
		ESLPassword:  dto.ESLPassword,
		SIPPort:      dto.SIPPort,
		MaxSessions:  dto.MaxSessions,
		Status:       dto.Status,
		Remark:       dto.Remark,
	}
	if fs.ESLPort == 0 {
		fs.ESLPort = 8021
	}
	if fs.SIPPort == 0 {
		fs.SIPPort = 5060
	}
	if fs.MaxSessions == 0 {
		fs.MaxSessions = 1000
	}

	err = s.systemRepo.CreateFS(ctx, fs)
	if err != nil {
		return nil, err
	}
	return fs, nil
}

// FSUpdate updates an FS instance.
func (s *SystemService) FSUpdate(ctx context.Context, dto *FSInstanceUpdateDTO) error {
	if dto.ID == 0 {
		return errors.ErrParamMissing
	}

	fs, err := s.systemRepo.GetFSByID(ctx, dto.ID)
	if err != nil {
		return err
	}

	// instance_code is the unique identifier and is not modified by updates; InstanceName is a display-only field that is not persisted.
	fs.Host = dto.Host
	fs.ESLPort = dto.ESLPort
	if dto.ESLPassword != "" {
		fs.ESLPassword = dto.ESLPassword
	}
	fs.SIPPort = dto.SIPPort
	fs.MaxSessions = dto.MaxSessions
	fs.Status = dto.Status
	fs.Remark = dto.Remark

	return s.systemRepo.UpdateFS(ctx, fs)
}

// FSDelete deletes an FS instance.
func (s *SystemService) FSDelete(ctx context.Context, id uint64) error {
	return s.systemRepo.DeleteFS(ctx, id)
}

// FSUpdateStatus updates an FS instance's status.
func (s *SystemService) FSUpdateStatus(ctx context.Context, id uint64, status int) error {
	return s.systemRepo.UpdateFSStatus(ctx, id, status)
}

// ===== Operation logs =====

// OperationLogPage returns a paginated list of operation logs.
func (s *SystemService) OperationLogPage(ctx context.Context, tenantID uint64, q *repository.OperationLogPageQuery) ([]*model.OperationLog, int64, error) {
	q.Page, q.PageSize = normalizePage(q.Page, q.PageSize)
	return s.systemRepo.PageQueryOperationLog(ctx, tenantID, q)
}

// CreateOperationLog creates an operation log.
func (s *SystemService) CreateOperationLog(ctx context.Context, log *model.OperationLog) error {
	return s.systemRepo.CreateOperationLog(ctx, log)
}
