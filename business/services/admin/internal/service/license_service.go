package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"go.uber.org/zap"

	"vocara/admin/internal/model"
	"vocara/admin/internal/repository"
	"vocara/pkg/errors"
	"vocara/pkg/license"
	"vocara/pkg/logger"
	pkgredis "vocara/pkg/redis"
)

// 全局 License 相关系统配置键（tenant_id=0，is_system=1）
const (
	configKeyInstallID = "license.install_id"
	configKeyLicense   = "license"
)

// LicenseService 授权管理服务：提供授权信息查询、License 导入热加载与重新加载。
// License 原文与安装标识主存系统配置表，经 Redis 分发给其余服务。
type LicenseService struct {
	manager *license.Manager
	repo    *repository.SystemRepository
}

// NewLicenseService 创建授权管理服务。repo 承载 License 原文与安装标识的读写。
func NewLicenseService(manager *license.Manager, repo *repository.SystemRepository) *LicenseService {
	return &LicenseService{manager: manager, repo: repo}
}

// LicenseInfo 授权信息视图：授权状态、载荷摘要、安装标识与本机机器码。
type LicenseInfo struct {
	// State 授权状态 valid/expiring/grace/expired/invalid
	State string `json:"state"`
	// Message 状态说明
	Message string `json:"message"`
	// DaysLeft 距到期整天数；永久为 -1，已过期为 0
	DaysLeft int `json:"days_left"`
	// LicenseID 授权编号
	LicenseID string `json:"license_id"`
	// Customer 客户名称
	Customer string `json:"customer"`
	// Edition 产品版本
	Edition string `json:"edition"`
	// LicenseType 授权类型 formal/trial
	LicenseType string `json:"license_type"`
	// IssuedAt 签发时间
	IssuedAt *time.Time `json:"issued_at,omitempty"`
	// NotBefore 生效时间
	NotBefore *time.Time `json:"not_before,omitempty"`
	// ExpireAt 到期时间
	ExpireAt *time.Time `json:"expire_at,omitempty"`
	// Perpetual 是否永久授权
	Perpetual bool `json:"perpetual"`
	// GraceDays 到期宽限天数
	GraceDays int `json:"grace_days"`
	// Modules 功能模块开关
	Modules map[string]bool `json:"modules"`
	// Limits 容量配额
	Limits license.Limits `json:"limits"`
	// Fingerprint 授权绑定信息
	Fingerprint license.Fingerprint `json:"fingerprint"`
	// InstallID 集群安装标识，客户凭此向厂商申请授权
	InstallID string `json:"install_id"`
	// LocalMachine 本机机器码，供媒体主机指纹采集核对
	LocalMachine string `json:"local_machine"`
}

// Info 返回当前授权信息；无有效载荷时仅填充状态、安装标识与本机机器码。
func (s *LicenseService) Info() *LicenseInfo {
	result := s.manager.Result()
	info := &LicenseInfo{
		State:        string(result.State),
		Message:      result.Message,
		DaysLeft:     result.DaysLeft,
		InstallID:    s.installID(context.Background()),
		LocalMachine: license.CollectFingerprint().Hash,
	}
	if claims := result.Claims; claims != nil {
		info.LicenseID = claims.LicenseID
		info.Customer = claims.Customer
		info.Edition = claims.Edition
		info.LicenseType = claims.LicenseType
		info.Perpetual = claims.IsPerpetual()
		info.GraceDays = claims.Grace()
		info.Modules = claims.Modules
		info.Limits = claims.Limits
		info.Fingerprint = claims.Fingerprint

		issuedAt := claims.IssuedAt
		notBefore := claims.NotBefore
		info.IssuedAt = &issuedAt
		info.NotBefore = &notBefore
		if claims.ExpireAt != nil {
			expireAt := *claims.ExpireAt
			info.ExpireAt = &expireAt
		}
	}
	return info
}

// Import 校验 License 内容，通过后写库、经 Redis 下发，并热加载到运行态。
func (s *LicenseService) Import(content []byte) error {
	if _, err := license.Parse(content, license.DefaultPublicKey()); err != nil {
		return errors.NewBizError(errors.CodeLicenseInvalid, errors.MsgLicenseInvalid).WithDetail(err.Error())
	}
	ctx := context.Background()
	if err := s.saveConfig(ctx, configKeyLicense, string(content), "License 授权原文"); err != nil {
		return errors.Wrap(err)
	}
	if err := s.setRedis(ctx, license.LicenseSourceKey, string(content)); err != nil {
		logger.Warn("publish license to redis failed", zap.Error(err))
	}
	if err := s.manager.ReloadFromBytes(content); err != nil {
		return errors.NewBizError(errors.CodeLicenseInvalid, errors.MsgLicenseInvalid).WithCause(err)
	}
	s.publishChange(ctx)
	return nil
}

// Reload 从系统配置表重新读取 License 原文，热加载到本进程并下发到 Redis。
// 下发 Redis 使其余服务在重启后能经 Redis 恢复授权，保证"数据库有 License 即持续有效"。
func (s *LicenseService) Reload() error {
	ctx := context.Background()
	content, err := s.getConfig(ctx, configKeyLicense)
	if err != nil {
		return errors.Wrap(err)
	}
	if content == "" {
		return errors.ErrLicenseMissing
	}
	if err := s.manager.ReloadFromBytes([]byte(content)); err != nil {
		return errors.ErrLicenseMissing.WithCause(err)
	}
	if err := s.setRedis(ctx, license.LicenseSourceKey, content); err != nil {
		logger.Warn("publish license to redis failed", zap.Error(err))
	}
	s.publishChange(ctx)
	return nil
}

// EnsureInstallID 确保集群安装标识存在：系统配置表缺失则生成并持久化；同时同步到 Redis 供其余服务读取。
func (s *LicenseService) EnsureInstallID(ctx context.Context) error {
	id, err := s.getConfig(ctx, configKeyInstallID)
	if err != nil {
		return err
	}
	if id == "" {
		generated, genErr := newInstallID()
		if genErr != nil {
			return genErr
		}
		id = generated
		if err := s.saveConfig(ctx, configKeyInstallID, id, "集群安装标识"); err != nil {
			return err
		}
		logger.Info("install id generated", zap.String("install_id", id))
	}
	if err := s.setRedis(ctx, license.InstallIDSourceKey, id); err != nil {
		logger.Warn("publish install id to redis failed", zap.Error(err))
	}
	return nil
}

// InstallID 实现 license.InstallIDProvider：从系统配置表读取安装标识。
// 返回空字符串表示尚未生成，交由管理器回退到配置值。
func (s *LicenseService) InstallID() (string, error) {
	return s.getConfig(context.Background(), configKeyInstallID)
}

// installID 读取安装标识，读取失败返回空字符串。
func (s *LicenseService) installID(ctx context.Context) string {
	id, err := s.getConfig(ctx, configKeyInstallID)
	if err != nil {
		return ""
	}
	return id
}

// getConfig 读取全局系统配置；不存在时返回空字符串。
func (s *LicenseService) getConfig(ctx context.Context, key string) (string, error) {
	cfg, err := s.repo.GetSystemConfig(ctx, 0, key)
	if err != nil {
		if bizErr, ok := errors.IsBizError(err); ok && bizErr.IsCode(errors.CodeNotFound) {
			return "", nil
		}
		return "", err
	}
	return cfg.ConfigValue, nil
}

// saveConfig 写入全局系统配置：已存在则更新，否则插入。
func (s *LicenseService) saveConfig(ctx context.Context, key, value, desc string) error {
	cfg, err := s.repo.GetSystemConfig(ctx, 0, key)
	if err != nil {
		if bizErr, ok := errors.IsBizError(err); !ok || !bizErr.IsCode(errors.CodeNotFound) {
			return err
		}
		cfg = &model.SystemConfig{
			TenantID:    0,
			ConfigKey:   key,
			ConfigValue: value,
			ConfigType:  "text",
			ConfigDesc:  desc,
			IsSystem:    1,
		}
		return s.repo.SaveSystemConfig(ctx, cfg)
	}
	cfg.ConfigValue = value
	cfg.ConfigDesc = desc
	return s.repo.SaveSystemConfig(ctx, cfg)
}

// setRedis 写入 Redis 分发 key；Redis 未就绪返回错误。
func (s *LicenseService) setRedis(ctx context.Context, key, value string) error {
	cli := pkgredis.Get()
	if cli == nil {
		return fmt.Errorf("redis not ready")
	}
	return cli.Set(ctx, key, value, 0)
}

// publishChange 广播授权变更通知，使其余服务立即复校，避免等待下个复校周期。
func (s *LicenseService) publishChange(ctx context.Context) {
	cli := pkgredis.Get()
	if cli == nil {
		return
	}
	if err := cli.Publish(ctx, license.LicenseChangedChannel, "changed"); err != nil {
		logger.Warn("publish license changed event failed", zap.Error(err))
	}
}

// newInstallID 生成 32 位十六进制安装标识（16 字节随机数）。
func newInstallID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
