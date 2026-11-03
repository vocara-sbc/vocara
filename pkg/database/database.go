// Package database provides a MySQL database wrapper based on GORM v2.
// It supports connection management, connection pool configuration, automatic migration, paginated queries, soft deletion, and context propagation.
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

// DB wraps a database client.
type DB struct {
	*gorm.DB
	config config.MySQLConfig
}

var (
	globalDB *DB
)

// New creates a database connection.
func New(cfg config.MySQLConfig) (*DB, error) {
	// Parse the log level
	logLevel := parseLogLevel(cfg.LogLevel)

	gormConfig := &gorm.Config{
		Logger: newZapGormLogger(logLevel), // Use the unified zap logger to avoid writing directly to stdout and creating a second set of log files
		NamingStrategy: schema.NamingStrategy{
			SingularTable: false, // Use plural table names
		},
		DisableForeignKeyConstraintWhenMigrating: true, // Do not create foreign key constraints during migration
	}

	// Open the connection
	db, err := gorm.Open(mysql.Open(cfg.DSN()), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("open mysql failed: %w", err)
	}

	// Get the underlying sql.DB to configure the connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql.DB failed: %w", err)
	}

	// Connection pool configuration
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Minute)
	sqlDB.SetConnMaxIdleTime(time.Duration(cfg.ConnMaxIdleTime) * time.Minute)

	// Test the connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping mysql failed: %w", err)
	}

	return &DB{
		DB:     db,
		config: cfg,
	}, nil
}

// Init initializes the global database connection.
func Init(cfg config.MySQLConfig) error {
	db, err := New(cfg)
	if err != nil {
		return err
	}
	globalDB = db
	return nil
}

// Get returns the global database connection.
func Get() *DB {
	return globalDB
}

// SetGlobal sets the global database for testing.
func SetGlobal(db *DB) {
	globalDB = db
}

// Close closes the database connection.
func (db *DB) Close() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// WithContext returns a DB with the context attached.
func (db *DB) WithContext(ctx context.Context) *gorm.DB {
	return db.DB.WithContext(ctx)
}

// AutoMigrate performs automatic migration.
func (db *DB) AutoMigrate(models ...interface{}) error {
	if !db.config.AutoMigrate {
		return nil
	}
	return db.DB.AutoMigrate(models...)
}

// parseLogLevel parses the GORM log level.
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

// ===== Transactions =====

// Transaction executes a transaction.
func (db *DB) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return db.WithContext(ctx).Transaction(fn)
}

// ===== Convenience query methods =====

// Create creates a record.
func (db *DB) Create(ctx context.Context, value interface{}) error {
	return db.WithContext(ctx).Create(value).Error
}

// Save saves a record.
func (db *DB) Save(ctx context.Context, value interface{}) error {
	return db.WithContext(ctx).Save(value).Error
}

// First returns the first record.
func (db *DB) First(ctx context.Context, dest interface{}, conds ...interface{}) error {
	return db.WithContext(ctx).First(dest, conds...).Error
}

// Find returns multiple records.
func (db *DB) Find(ctx context.Context, dest interface{}, conds ...interface{}) (int64, error) {
	tx := db.WithContext(ctx).Find(dest, conds...)
	return tx.RowsAffected, tx.Error
}

// Update updates a single field.
func (db *DB) Update(ctx context.Context, model interface{}, column string, value interface{}) error {
	return db.WithContext(ctx).Model(model).Update(column, value).Error
}

// Updates updates multiple fields.
func (db *DB) Updates(ctx context.Context, model interface{}, values interface{}) error {
	return db.WithContext(ctx).Model(model).Updates(values).Error
}

// Delete deletes records.
func (db *DB) Delete(ctx context.Context, value interface{}, conds ...interface{}) error {
	return db.WithContext(ctx).Delete(value, conds...).Error
}

// Count counts records.
func (db *DB) Count(ctx context.Context, model interface{}) (int64, error) {
	var count int64
	err := db.WithContext(ctx).Model(model).Count(&count).Error
	return count, err
}
