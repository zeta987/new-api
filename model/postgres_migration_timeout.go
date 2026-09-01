package model

import (
	"fmt"

	"gorm.io/gorm"
)

const (
	postgresMigrationLockTimeout      = "5s"
	postgresMigrationStatementTimeout = "30s"
)

func configurePostgresMigrationTimeouts(tx *gorm.DB) error {
	if tx == nil {
		return fmt.Errorf("configure postgres migration timeouts: database is nil")
	}
	if err := tx.Exec(
		"SELECT set_config('lock_timeout', ?, true)",
		postgresMigrationLockTimeout,
	).Error; err != nil {
		return fmt.Errorf("configure postgres migration lock timeout: %w", err)
	}
	if err := tx.Exec(
		"SELECT set_config('statement_timeout', ?, true)",
		postgresMigrationStatementTimeout,
	).Error; err != nil {
		return fmt.Errorf("configure postgres migration statement timeout: %w", err)
	}
	return nil
}
