package service

import (
	"context"

	"vocara/admin/internal/model"
	"vocara/admin/internal/repository"
	"vocara/pkg/errors"
)

// SystemService 系统管理服务
type SystemService struct {
	systemRepo *repository.SystemRepository
}

// NewSystemService 创建系统管理服务
func NewSystemService() *SystemService {
	return &SystemService{
		systemRepo: repository.NewSystemRepository(),
	}
}

// ===== 系统配置 =====

// GetConfigList 获取系统配置列表
func (s *SystemService) GetConfigList(ctx context.Context, tenantID uint64) ([]*model.SystemConfig, error) {
	return s.systemRepo.ListSystemConfig(ctx, tenantID)
}

// GetConfig 获取系统配置
func (s *SystemService) GetConfig(ctx context.Context, tenantID uint64, key string) (*model.SystemConfig, error) {
	return s.systemRepo.GetSystemConfig(ctx, tenantID, key)
}

// SaveConfig 保存系统配置
func (s *SystemService) SaveConfig(ctx context.Context, config *model.SystemConfig) error {
	if config.ConfigKey == "" {
		return errors.ErrParamMissing
	}
	return s.systemRepo.SaveSystemConfig(ctx, config)
}

// DeleteConfig 删除系统配置
func (s *SystemService) DeleteConfig(ctx context.Context, id uint64) error {
	return s.systemRepo.DeleteSystemConfig(ctx, id)
}

// ===== FS 实例管理 =====

// FSInstanceSaveDTO FS实例保存
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

// FSInstanceUpdateDTO FS实例更新
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

// FSPage 分页查询FS实例
func (s *SystemService) FSPage(ctx context.Context, q *repository.FSPageQuery) ([]*model.FSInstance, int64, error) {
	q.Page, q.PageSize = normalizePage(q.Page, q.PageSize)
	return s.systemRepo.PageQueryFS(ctx, q)
}

// FSList 查询所有FS实例
func (s *SystemService) FSList(ctx context.Context) ([]*model.FSInstance, error) {
	return s.systemRepo.ListAllFS(ctx)
}

// FSGet 获取FS实例详情
func (s *SystemService) FSGet(ctx context.Context, id uint64) (*model.FSInstance, error) {
	return s.systemRepo.GetFSByID(ctx, id)
}

// FSCreate 创建FS实例
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

// FSUpdate 更新FS实例
func (s *SystemService) FSUpdate(ctx context.Context, dto *FSInstanceUpdateDTO) error {
	if dto.ID == 0 {
		return errors.ErrParamMissing
	}

	fs, err := s.systemRepo.GetFSByID(ctx, dto.ID)
	if err != nil {
		return err
	}

	// instance_code 为唯一标识，更新接口不修改编码；InstanceName 为展示字段、不落库
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

// FSDelete 删除FS实例
func (s *SystemService) FSDelete(ctx context.Context, id uint64) error {
	return s.systemRepo.DeleteFS(ctx, id)
}

// FSUpdateStatus 更新FS实例状态
func (s *SystemService) FSUpdateStatus(ctx context.Context, id uint64, status int) error {
	return s.systemRepo.UpdateFSStatus(ctx, id, status)
}

// ===== 操作日志 =====

// OperationLogPage 分页查询操作日志
func (s *SystemService) OperationLogPage(ctx context.Context, tenantID uint64, q *repository.OperationLogPageQuery) ([]*model.OperationLog, int64, error) {
	q.Page, q.PageSize = normalizePage(q.Page, q.PageSize)
	return s.systemRepo.PageQueryOperationLog(ctx, tenantID, q)
}

// CreateOperationLog 创建操作日志
func (s *SystemService) CreateOperationLog(ctx context.Context, log *model.OperationLog) error {
	return s.systemRepo.CreateOperationLog(ctx, log)
}
