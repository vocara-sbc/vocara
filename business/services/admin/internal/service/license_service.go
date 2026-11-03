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

// Global license system configuration keys (tenant_id=0, is_system=1).
const (
	configKeyInstallID = "license.install_id"
	configKeyLicense   = "license"
)

// LicenseService provides license information, import with hot loading, and reload operations.
// The system configuration table stores license content and the installation ID, and Redis distributes them to other services.
type LicenseService struct {
	manager *license.Manager
	repo    *repository.SystemRepository
}

// NewLicenseService creates a license service whose repository reads and writes license content and the installation ID.
func NewLicenseService(manager *license.Manager, repo *repository.SystemRepository) *LicenseService {
	return &LicenseService{manager: manager, repo: repo}
}

// LicenseInfo contains the license status, claims summary, installation ID, and local machine fingerprint.
type LicenseInfo struct {
	// State is the license status: valid, expiring, grace, expired, or invalid.
	State string `json:"state"`
	// Message describes the license status.
	Message string `json:"message"`
	// DaysLeft is the whole days until expiration, -1 for perpetual licenses, or 0 when expired.
	DaysLeft int `json:"days_left"`
	// LicenseID is the license identifier.
	LicenseID string `json:"license_id"`
	// Customer is the customer name.
	Customer string `json:"customer"`
	// Edition is the product edition.
	Edition string `json:"edition"`
	// LicenseType is formal or trial.
	LicenseType string `json:"license_type"`
	// IssuedAt is the issue time.
	IssuedAt *time.Time `json:"issued_at,omitempty"`
	// NotBefore is the activation time.
	NotBefore *time.Time `json:"not_before,omitempty"`
	// ExpireAt is the expiration time.
	ExpireAt *time.Time `json:"expire_at,omitempty"`
	// Perpetual indicates a perpetual license.
	Perpetual bool `json:"perpetual"`
	// GraceDays is the number of grace days after expiration.
	GraceDays int `json:"grace_days"`
	// Modules contains feature module switches.
	Modules map[string]bool `json:"modules"`
	// Limits contains capacity quotas.
	Limits license.Limits `json:"limits"`
	// Fingerprint contains license binding information.
	Fingerprint license.Fingerprint `json:"fingerprint"`
	// InstallID identifies the cluster installation when requesting a license from the vendor.
	InstallID string `json:"install_id"`
	// LocalMachine is the local machine fingerprint for verifying media host fingerprint collection.
	LocalMachine string `json:"local_machine"`
}

// Info returns current license information; without valid claims it includes only status, installation ID, and local machine fingerprint.
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

// Import validates license content, saves it, distributes it through Redis, and hot-loads it into the running process.
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

// Reload reads license content from the system configuration table, hot-loads it, and distributes it through Redis.
// Redis distribution lets other services restore the license after restarting whenever the database contains one.
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

// EnsureInstallID generates and persists a missing cluster installation ID and synchronizes it to Redis for other services.
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

// InstallID implements license.InstallIDProvider by reading the installation ID from the system configuration table.
// An empty string lets the manager use the configured value.
func (s *LicenseService) InstallID() (string, error) {
	return s.getConfig(context.Background(), configKeyInstallID)
}

// installID reads the installation ID and returns an empty string on failure.
func (s *LicenseService) installID(ctx context.Context) string {
	id, err := s.getConfig(ctx, configKeyInstallID)
	if err != nil {
		return ""
	}
	return id
}

// getConfig reads a global system configuration and returns an empty string when it does not exist.
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

// saveConfig updates an existing global system configuration or inserts it.
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

// setRedis writes a Redis distribution key and returns an error when Redis is unavailable.
func (s *LicenseService) setRedis(ctx context.Context, key, value string) error {
	cli := pkgredis.Get()
	if cli == nil {
		return fmt.Errorf("redis not ready")
	}
	return cli.Set(ctx, key, value, 0)
}

// publishChange broadcasts a license change so other services revalidate immediately.
func (s *LicenseService) publishChange(ctx context.Context) {
	cli := pkgredis.Get()
	if cli == nil {
		return
	}
	if err := cli.Publish(ctx, license.LicenseChangedChannel, "changed"); err != nil {
		logger.Warn("publish license changed event failed", zap.Error(err))
	}
}

// newInstallID generates a 32-character hexadecimal installation ID from 16 random bytes.
func newInstallID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
