package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB wraps a GORM connection to ~/.nexa/state.db.
type DB struct {
	*gorm.DB
	home string
}

// StateFile returns the state.db path under home.
func StateFile(home string) string {
	return filepath.Join(home, "state.db")
}

// Open opens (or creates) state.db and runs AutoMigrate.
func Open(home string) (*DB, error) {
	if err := os.MkdirAll(home, 0o755); err != nil {
		return nil, fmt.Errorf("create home: %w", err)
	}
	path := StateFile(home)
	created := false
	if _, err := os.Stat(path); os.IsNotExist(err) {
		created = true
	}

	gdb, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("open state.db: %w", err)
	}
	if created {
		_ = os.Chmod(path, 0o600)
	}

	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, fmt.Errorf("sql db: %w", err)
	}
	if _, err := sqlDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("pragma foreign_keys: %w", err)
	}

	db := &DB{DB: gdb, home: home}
	if err := db.autoMigrate(); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	if err := db.ensureAdminUser(); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	if err := db.backfillAPIKeyDefaults(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("backfill api key defaults: %w", err)
	}
	if err := db.MigrateFromFiles(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("migrate from files: %w", err)
	}
	return db, nil
}

// Home returns the config home directory.
func (db *DB) Home() string { return db.home }

// Close closes the underlying SQL connection.
func (db *DB) Close() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (db *DB) autoMigrate() error {
	if err := db.AutoMigrate(
		&User{},
		&APIKey{},
		&Meta{},
		&Setting{},
		&ProviderRow{},
		&Credential{},
		&MCPServerRow{},
		&SessionRow{},
		&KanbanRow{},
	); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}
	return nil
}

// ensureAdminUser promotes the earliest user when no admin exists yet.
// An empty users table is valid; the first RegisterUser becomes admin.
func (db *DB) ensureAdminUser() error {
	var n int64
	if err := db.Model(&User{}).Where("role = ?", RoleAdmin).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	var u User
	err := db.Order("created_at asc").First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("find user to promote: %w", err)
	}
	if err := db.Model(&User{}).Where("id = ?", u.ID).Update("role", RoleAdmin).Error; err != nil {
		return fmt.Errorf("promote user %q to admin: %w", u.ID, err)
	}
	return nil
}

// backfillAPIKeyDefaults repairs rows created before scopes/enabled existed.
func (db *DB) backfillAPIKeyDefaults() error {
	return db.Model(&APIKey{}).
		Where("scopes IS NULL OR scopes = ''").
		Update("scopes", "*").Error
}
