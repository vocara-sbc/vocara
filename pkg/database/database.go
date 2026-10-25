// Package database 提供基于 GORM v2 的 MySQL 数据库封装
// 支持连接管理、连接池配置、自动迁移、分页查询、软删除、上下文透传
package database

import (
	"context"
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"

	"vocara/pkg/config"
)

// DB 数据库客户端封装
type DB struct {
	*gorm.DB
	config config.MySQLConfig
}

var (
	globalDB *DB
)

// New 创建新的数据库连接
func New(cfg config.MySQLConfig) (*DB, error) {
	// 解析日志级别
	logLevel := parseLogLevel(cfg.LogLevel)

	gormConfig := &gorm.Config{
		Logger: newZapGormLogger(logLevel), // 接入统一 zap 日志，避免直写 stdout 产生第二套日志文件
		NamingStrategy: schema.NamingStrategy{
			SingularTable: false, // 使用复数表名
		},
		DisableForeignKeyConstraintWhenMigrating: true, // 迁移时不创建外键约束
	}

	// 打开连接
	db, err := gorm.Open(mysql.Open(cfg.DSN()), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("open mysql failed: %w", err)
	}

	// 获取底层 sql.DB 配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql.DB failed: %w", err)
	}

	// 连接池配置
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Minute)
	sqlDB.SetConnMaxIdleTime(time.Duration(cfg.ConnMaxIdleTime) * time.Minute)

	// 测试连接
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping mysql failed: %w", err)
	}

	return &DB{
		DB:     db,
		config: cfg,
	}, nil
}

// Init 初始化全局数据库连接
func Init(cfg config.MySQLConfig) error {
	db, err := New(cfg)
	if err != nil {
		return err
	}
	globalDB = db
	return nil
}

// Get 获取全局数据库连接
func Get() *DB {
	return globalDB
}

// SetGlobal 设置全局数据库（用于测试）
func SetGlobal(db *DB) {
	globalDB = db
}

// Close 关闭数据库连接
func (db *DB) Close() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// WithContext 返回带上下文的 DB
func (db *DB) WithContext(ctx context.Context) *gorm.DB {
	return db.DB.WithContext(ctx)
}

// AutoMigrate 自动迁移
func (db *DB) AutoMigrate(models ...interface{}) error {
	if !db.config.AutoMigrate {
		return nil
	}
	return db.DB.AutoMigrate(models...)
}

// parseLogLevel 解析 GORM 日志级别
func parseLogLevel(level string) logger.LogLevel {
	switch level {
	case "silent":
		return logger.Silent
	case "error":
		return logger.Error
	case "warn":
		return logger.Warn
	case "info":
		return logger.Info
	default:
		return logger.Warn
	}
}

// ===== 事务 =====

// Transaction 执行事务
func (db *DB) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return db.WithContext(ctx).Transaction(fn)
}

// ===== 便捷查询方法 =====

// Create 创建记录
func (db *DB) Create(ctx context.Context, value interface{}) error {
	return db.WithContext(ctx).Create(value).Error
}

// Save 保存记录
func (db *DB) Save(ctx context.Context, value interface{}) error {
	return db.WithContext(ctx).Save(value).Error
}

// First 获取第一条记录
func (db *DB) First(ctx context.Context, dest interface{}, conds ...interface{}) error {
	return db.WithContext(ctx).First(dest, conds...).Error
}

// Find 获取多条记录
func (db *DB) Find(ctx context.Context, dest interface{}, conds ...interface{}) (int64, error) {
	tx := db.WithContext(ctx).Find(dest, conds...)
	return tx.RowsAffected, tx.Error
}

// Update 更新单个字段
func (db *DB) Update(ctx context.Context, model interface{}, column string, value interface{}) error {
	return db.WithContext(ctx).Model(model).Update(column, value).Error
}

// Updates 更新多个字段
func (db *DB) Updates(ctx context.Context, model interface{}, values interface{}) error {
	return db.WithContext(ctx).Model(model).Updates(values).Error
}

// Delete 删除记录
func (db *DB) Delete(ctx context.Context, value interface{}, conds ...interface{}) error {
	return db.WithContext(ctx).Delete(value, conds...).Error
}

// Count 统计记录数
func (db *DB) Count(ctx context.Context, model interface{}) (int64, error) {
	var count int64
	err := db.WithContext(ctx).Model(model).Count(&count).Error
	return count, err
}
