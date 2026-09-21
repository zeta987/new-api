package model

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

const (
	postgresNamedUniqueMigrationTable      = "postgres_named_unique_override_records"
	postgresNamedUniqueMigrationIndex      = "uk_postgres_named_unique_migration_key_deleted_at"
	postgresNamedUniqueMigrationConstraint = "idx_postgres_named_unique_migrations_key"
	postgresIdentifierLengthMigrationTable = "postgres_dialector_identifier_length_migration_records_test"
)

type postgresNamedUniqueMigration struct {
	ID        int
	Key       string         `gorm:"size:64;not null;uniqueIndex:uk_postgres_named_unique_migration_key_deleted_at,priority:1"`
	DeletedAt gorm.DeletedAt `gorm:"uniqueIndex:uk_postgres_named_unique_migration_key_deleted_at,priority:2"`
}

type postgresIdentifierLengthMigration struct {
	ID   int
	Name string `gorm:"size:64;not null;uniqueIndex"`
}

func openPostgresMigrationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("TEST_POSTGRES_DSN"))
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN is not configured")
	}
	t.Setenv("SQL_DSN", dsn)

	db, dbType, err := chooseDB("SQL_DSN", false)
	require.NoError(t, err)
	assert.Equal(t, common.DatabaseTypePostgreSQL, dbType)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	return db
}

func createPostgresMigrationTestSchema(t *testing.T, db *gorm.DB, prefix string) string {
	t.Helper()
	schemaName := fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	require.NoError(t, db.Exec(
		"CREATE SCHEMA ?",
		clause.Table{Name: schemaName},
	).Error)
	t.Cleanup(func() {
		require.NoError(t, db.Exec(
			"DROP SCHEMA ? CASCADE",
			clause.Table{Name: schemaName},
		).Error)
	})
	return schemaName
}

func bindPostgresMigrationTestTable(db *gorm.DB, schemaName, tableName string) (*gorm.DB, error) {
	if err := db.Exec(
		"SET LOCAL search_path TO ?",
		clause.Table{Name: schemaName},
	).Error; err != nil {
		return nil, err
	}
	return db.Table(tableName), nil
}

func beginPostgresMigrationTestSchema(t *testing.T, db *gorm.DB, prefix, tableName string) (*gorm.DB, string) {
	t.Helper()
	schemaName := createPostgresMigrationTestSchema(t, db, prefix)
	tx := db.Begin()
	require.NoError(t, tx.Error)
	t.Cleanup(func() { require.NoError(t, tx.Rollback().Error) })
	tableDB, err := bindPostgresMigrationTestTable(tx, schemaName, tableName)
	require.NoError(t, err)
	return tableDB, schemaName
}

func TestPostgresMigrationTestTableUsesLocalSearchPath(t *testing.T) {
	recorder := &migrationSQLRecorder{}
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  "host=127.0.0.1 user=unused dbname=unused sslmode=disable",
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		DryRun:               true,
		DisableAutomaticPing: true,
		Logger:               recorder,
	})
	require.NoError(t, err)

	tableDB, err := bindPostgresMigrationTestTable(
		db,
		"isolated_schema",
		postgresNamedUniqueMigrationTable,
	)
	require.NoError(t, err)
	require.NoError(t, tableDB.Migrator().DropConstraint(
		&postgresNamedUniqueMigration{},
		postgresNamedUniqueMigrationConstraint,
	))

	recorder.mu.Lock()
	statements := append([]string(nil), recorder.statements...)
	recorder.mu.Unlock()
	require.Len(t, statements, 2)
	assert.Equal(t, `SET LOCAL search_path TO "isolated_schema"`, statements[0])
	assert.Equal(t,
		`ALTER TABLE "postgres_named_unique_override_records" DROP CONSTRAINT "idx_postgres_named_unique_migrations_key"`,
		statements[1],
	)
	assert.NotContains(t, statements[1], "isolated_schema")
}

func TestConfigurePostgresMigrationTimeouts(t *testing.T) {
	db := openPostgresMigrationTestDB(t)
	type postgresTimeoutSettings struct {
		LockTimeout      string `gorm:"column:lock_timeout"`
		StatementTimeout string `gorm:"column:statement_timeout"`
	}
	originalSessionSettings := postgresTimeoutSettings{}
	baselineSessionSettings := postgresTimeoutSettings{
		LockTimeout:      "1s",
		StatementTimeout: "2s",
	}
	var insideTransaction postgresTimeoutSettings
	var afterTransaction postgresTimeoutSettings

	require.NoError(t, db.Connection(func(connection *gorm.DB) error {
		if err := connection.Raw(`
SELECT current_setting('lock_timeout') AS lock_timeout,
       current_setting('statement_timeout') AS statement_timeout`).Scan(&originalSessionSettings).Error; err != nil {
			return err
		}
		defer func() {
			require.NoError(t, connection.Exec(
				"SELECT set_config('lock_timeout', ?, false)",
				originalSessionSettings.LockTimeout,
			).Error)
			require.NoError(t, connection.Exec(
				"SELECT set_config('statement_timeout', ?, false)",
				originalSessionSettings.StatementTimeout,
			).Error)
		}()
		if err := connection.Exec(
			"SELECT set_config('lock_timeout', ?, false)",
			baselineSessionSettings.LockTimeout,
		).Error; err != nil {
			return err
		}
		if err := connection.Exec(
			"SELECT set_config('statement_timeout', ?, false)",
			baselineSessionSettings.StatementTimeout,
		).Error; err != nil {
			return err
		}
		if err := connection.Transaction(func(tx *gorm.DB) error {
			if err := configurePostgresMigrationTimeouts(tx); err != nil {
				return err
			}
			return tx.Raw(`
SELECT current_setting('lock_timeout') AS lock_timeout,
       current_setting('statement_timeout') AS statement_timeout`).Scan(&insideTransaction).Error
		}); err != nil {
			return err
		}
		return connection.Raw(`
SELECT current_setting('lock_timeout') AS lock_timeout,
       current_setting('statement_timeout') AS statement_timeout`).Scan(&afterTransaction).Error
	}))
	assert.Equal(t, "5s", insideTransaction.LockTimeout)
	assert.Equal(t, "30s", insideTransaction.StatementTimeout)
	assert.Equal(t, baselineSessionSettings, afterTransaction)
}

func TestPostgresUniquenessMigrationsBoundLockWait(t *testing.T) {
	db := openPostgresMigrationTestDB(t)
	tests := []struct {
		name      string
		model     any
		tableName string
		prepare   func(*testing.T, *gorm.DB)
		migrate   func(*gorm.DB) error
	}{
		{
			name:      "token",
			model:     &Token{},
			tableName: "tokens",
			prepare: func(t *testing.T, db *gorm.DB) {
				t.Helper()
				require.NoError(t, db.Exec(
					"ALTER TABLE ? ADD CONSTRAINT ? UNIQUE (?)",
					clause.Table{Name: "tokens"},
					clause.Column{Name: gormTokenKeyConstraint},
					clause.Column{Name: "key"},
				).Error)
			},
			migrate: migrateTokenKeyUniqueness,
		},
		{
			name:      "prefill_group",
			model:     &PrefillGroup{},
			tableName: "prefill_groups",
			prepare: func(t *testing.T, db *gorm.DB) {
				t.Helper()
				require.NoError(t, db.Migrator().DropIndex(&PrefillGroup{}, prefillGroupNameIndex))
				require.NoError(t, db.Exec(
					"ALTER TABLE ? ADD CONSTRAINT ? UNIQUE (?)",
					clause.Table{Name: "prefill_groups"},
					clause.Column{Name: legacyPrefillGroupNameUnique},
					clause.Column{Name: "name"},
				).Error)
			},
			migrate: migratePrefillGroupUniqueness,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			schemaName := fmt.Sprintf("postgres_timeout_%s_%d", test.name, time.Now().UnixNano())
			require.NoError(t, db.Exec(
				"CREATE SCHEMA ?",
				clause.Table{Name: schemaName},
			).Error)
			t.Cleanup(func() {
				require.NoError(t, db.Exec(
					"DROP SCHEMA ? CASCADE",
					clause.Table{Name: schemaName},
				).Error)
			})

			setupTx := db.Begin()
			require.NoError(t, setupTx.Error)
			t.Cleanup(func() { _ = setupTx.Rollback().Error })
			require.NoError(t, setupTx.Exec(
				"SET LOCAL search_path TO ?",
				clause.Table{Name: schemaName},
			).Error)
			require.NoError(t, setupTx.AutoMigrate(test.model))
			test.prepare(t, setupTx)
			require.NoError(t, setupTx.Commit().Error)

			lockHolder := db.Begin()
			require.NoError(t, lockHolder.Error)
			t.Cleanup(func() { _ = lockHolder.Rollback().Error })
			require.NoError(t, lockHolder.Exec(
				"SET LOCAL search_path TO ?",
				clause.Table{Name: schemaName},
			).Error)
			require.NoError(t, lockHolder.Exec(
				"LOCK TABLE ? IN ACCESS EXCLUSIVE MODE",
				clause.Table{Name: test.tableName},
			).Error)

			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()
			migrationTx := db.WithContext(ctx).Begin()
			require.NoError(t, migrationTx.Error)
			t.Cleanup(func() { _ = migrationTx.Rollback().Error })
			require.NoError(t, migrationTx.Exec(
				"SET LOCAL search_path TO ?",
				clause.Table{Name: schemaName},
			).Error)
			err := test.migrate(migrationTx)
			require.Error(t, err)
			require.Contains(t, err.Error(), "SQLSTATE 55P03")
			require.NoError(t, migrationTx.Rollback().Error)
			require.NoError(t, lockHolder.Rollback().Error)
		})
	}
}

func TestMigratePrefillGroupUniquenessContractWaitsForLockAndReplacesLegacy(t *testing.T) {
	db := openPostgresMigrationTestDB(t)
	schemaName := fmt.Sprintf("postgres_contract_prefill_%d", time.Now().UnixNano())
	require.NoError(t, db.Exec(
		"CREATE SCHEMA ?",
		clause.Table{Name: schemaName},
	).Error)
	t.Cleanup(func() {
		require.NoError(t, db.Exec(
			"DROP SCHEMA ? CASCADE",
			clause.Table{Name: schemaName},
		).Error)
	})

	setupTx := db.Begin()
	require.NoError(t, setupTx.Error)
	t.Cleanup(func() { _ = setupTx.Rollback().Error })
	require.NoError(t, setupTx.Exec(
		"SET LOCAL search_path TO ?",
		clause.Table{Name: schemaName},
	).Error)
	require.NoError(t, setupTx.AutoMigrate(&PrefillGroup{}))
	original := PrefillGroup{
		Name:        "contract-shared-name",
		Type:        "model",
		Items:       JSONValue(`[]`),
		Description: "preserve through lock timeout",
	}
	require.NoError(t, setupTx.Create(&original).Error)
	require.NoError(t, setupTx.Exec(
		"ALTER TABLE ? ADD CONSTRAINT ? UNIQUE (?)",
		clause.Table{Name: "prefill_groups"},
		clause.Column{Name: legacyPrefillGroupNameUnique},
		clause.Column{Name: "name"},
	).Error)
	assert.True(t, setupTx.Migrator().HasColumn(&PrefillGroup{}, "DeletedAt"))
	assert.True(t, setupTx.Migrator().HasConstraint(&PrefillGroup{}, legacyPrefillGroupNameUnique))
	targetIndex, err := inspectPrefillGroupNameIndex(setupTx, "prefill_groups")
	require.NoError(t, err)
	assert.True(t, targetIndex.valid)
	require.NoError(t, setupTx.Commit().Error)

	lockHolder := db.Begin()
	require.NoError(t, lockHolder.Error)
	t.Cleanup(func() { _ = lockHolder.Rollback().Error })
	require.NoError(t, lockHolder.Exec(
		"SET LOCAL search_path TO ?",
		clause.Table{Name: schemaName},
	).Error)
	require.NoError(t, lockHolder.Exec(
		"LOCK TABLE ? IN ACCESS SHARE MODE",
		clause.Table{Name: "prefill_groups"},
	).Error)

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	migrationTx := db.WithContext(ctx).Begin()
	require.NoError(t, migrationTx.Error)
	t.Cleanup(func() { _ = migrationTx.Rollback().Error })
	require.NoError(t, migrationTx.Exec(
		"SET LOCAL search_path TO ?",
		clause.Table{Name: schemaName},
	).Error)
	migrationErr := migratePrefillGroupUniqueness(migrationTx)
	require.Error(t, migrationErr)
	require.Contains(t, migrationErr.Error(), "SQLSTATE 55P03")
	require.NoError(t, migrationTx.Rollback().Error)

	blockedStateTx := db.Begin()
	require.NoError(t, blockedStateTx.Error)
	t.Cleanup(func() { _ = blockedStateTx.Rollback().Error })
	require.NoError(t, blockedStateTx.Exec(
		"SET LOCAL search_path TO ?",
		clause.Table{Name: schemaName},
	).Error)
	assert.True(t, blockedStateTx.Migrator().HasConstraint(&PrefillGroup{}, legacyPrefillGroupNameUnique))
	targetIndex, err = inspectPrefillGroupNameIndex(blockedStateTx, "prefill_groups")
	require.NoError(t, err)
	assert.True(t, targetIndex.valid)
	var preserved PrefillGroup
	require.NoError(t, blockedStateTx.First(&preserved, original.Id).Error)
	assert.Equal(t, original.Name, preserved.Name)
	assert.Equal(t, original.Description, preserved.Description)
	var blockedTotal int64
	require.NoError(t, blockedStateTx.Unscoped().Model(&PrefillGroup{}).Count(&blockedTotal).Error)
	assert.EqualValues(t, 1, blockedTotal)
	require.NoError(t, blockedStateTx.Rollback().Error)

	require.NoError(t, lockHolder.Rollback().Error)

	successTx := db.Begin()
	require.NoError(t, successTx.Error)
	t.Cleanup(func() { _ = successTx.Rollback().Error })
	require.NoError(t, successTx.Exec(
		"SET LOCAL search_path TO ?",
		clause.Table{Name: schemaName},
	).Error)
	require.NoError(t, migratePrefillGroupUniqueness(successTx))
	require.NoError(t, successTx.Commit().Error)

	verifyTx := db.Begin()
	require.NoError(t, verifyTx.Error)
	t.Cleanup(func() { _ = verifyTx.Rollback().Error })
	require.NoError(t, verifyTx.Exec(
		"SET LOCAL search_path TO ?",
		clause.Table{Name: schemaName},
	).Error)
	assert.False(t, verifyTx.Migrator().HasConstraint(&PrefillGroup{}, legacyPrefillGroupNameUnique))
	targetIndex, err = inspectPrefillGroupNameIndex(verifyTx, "prefill_groups")
	require.NoError(t, err)
	assert.True(t, targetIndex.valid)
	require.NoError(t, verifyTx.First(&preserved, original.Id).Error)
	assert.Equal(t, original.Name, preserved.Name)
	assert.Equal(t, original.Description, preserved.Description)

	duplicateErr := verifyTx.Transaction(func(tx *gorm.DB) error {
		return tx.Create(&PrefillGroup{
			Name:  original.Name,
			Type:  "model",
			Items: JSONValue(`[]`),
		}).Error
	})
	require.Error(t, duplicateErr)
	require.NoError(t, verifyTx.Delete(&preserved).Error)
	reuseErr := verifyTx.Transaction(func(tx *gorm.DB) error {
		return tx.Create(&PrefillGroup{
			Name:  original.Name,
			Type:  "model",
			Items: JSONValue(`[]`),
		}).Error
	})
	require.NoError(t, reuseErr)
	var total, active int64
	require.NoError(t, verifyTx.Unscoped().Model(&PrefillGroup{}).
		Where("name = ?", original.Name).Count(&total).Error)
	require.NoError(t, verifyTx.Model(&PrefillGroup{}).
		Where("name = ?", original.Name).Count(&active).Error)
	assert.EqualValues(t, 2, total)
	assert.EqualValues(t, 1, active)
	require.NoError(t, verifyTx.Rollback().Error)
}

func TestChooseDBPostgreSQLRepeatedAutoMigrateNamedUniqueIndex(t *testing.T) {
	db := openPostgresMigrationTestDB(t)
	externalSchema := createPostgresMigrationTestSchema(t, db, "postgres_named_unique_external")
	externalTableName := externalSchema + "." + postgresNamedUniqueMigrationTable
	externalTableDB := db.Table(externalTableName)
	require.NoError(t, externalTableDB.AutoMigrate(&postgresNamedUniqueMigration{}))
	require.NoError(t, db.Exec(
		"ALTER TABLE ? ADD CONSTRAINT ? UNIQUE (?)",
		clause.Table{Name: externalTableName},
		clause.Column{Name: postgresNamedUniqueMigrationConstraint},
		clause.Column{Name: "key"},
	).Error)
	require.NoError(t, externalTableDB.Create(&postgresNamedUniqueMigration{Key: "external-preserved"}).Error)

	tableDB, isolatedSchema := beginPostgresMigrationTestSchema(
		t,
		db,
		"postgres_named_unique_isolated",
		postgresNamedUniqueMigrationTable,
	)

	require.NoError(t, tableDB.AutoMigrate(&postgresNamedUniqueMigration{}))
	require.NoError(t, tableDB.Exec(
		"ALTER TABLE ? ADD CONSTRAINT ? UNIQUE (?)",
		clause.Table{Name: postgresNamedUniqueMigrationTable},
		clause.Column{Name: postgresNamedUniqueMigrationConstraint},
		clause.Column{Name: "key"},
	).Error)
	// The regression this guards is that a second AutoMigrate must not abort with
	// SQLSTATE 42704 by dropping a constraint name GORM merely inferred. It must
	// still succeed here. Since rc.38 the migrator resolves single-column
	// uniqueness from pg_catalog, so a stale constraint on a column the model now
	// covers with a composite uniqueIndex is reconciled away instead of being left
	// behind. This mirrors the production schema, where models.model_name and
	// vendors.name carry the same legacy constraint alongside their delete-aware
	// unique indexes.
	require.NoError(t, tableDB.AutoMigrate(&postgresNamedUniqueMigration{}))

	var isolatedConstraintCount int64
	require.NoError(t, tableDB.Raw(`
SELECT count(*)
FROM information_schema.table_constraints
WHERE table_schema = ? AND table_name = ? AND constraint_name = ?`,
		isolatedSchema,
		postgresNamedUniqueMigrationTable,
		postgresNamedUniqueMigrationConstraint,
	).Scan(&isolatedConstraintCount).Error)
	assert.Zero(t, isolatedConstraintCount)

	var isolatedIndexCount int64
	require.NoError(t, tableDB.Raw(`
SELECT count(*)
FROM pg_catalog.pg_indexes
WHERE schemaname = ? AND tablename = ? AND indexname = ?`,
		isolatedSchema,
		postgresNamedUniqueMigrationTable,
		postgresNamedUniqueMigrationIndex,
	).Scan(&isolatedIndexCount).Error)
	assert.EqualValues(t, 1, isolatedIndexCount)

	var externalConstraintCount int64
	require.NoError(t, db.Raw(`
SELECT count(*)
FROM information_schema.table_constraints
WHERE table_schema = ? AND table_name = ? AND constraint_name = ?`,
		externalSchema,
		postgresNamedUniqueMigrationTable,
		postgresNamedUniqueMigrationConstraint,
	).Scan(&externalConstraintCount).Error)
	assert.EqualValues(t, 1, externalConstraintCount)

	var externalRows int64
	require.NoError(t, externalTableDB.Model(&postgresNamedUniqueMigration{}).Count(&externalRows).Error)
	assert.EqualValues(t, 1, externalRows)
}

func TestChooseDBPostgreSQLRetainsSavepointSupport(t *testing.T) {
	db := openPostgresMigrationTestDB(t)
	tableDB, _ := beginPostgresMigrationTestSchema(
		t,
		db,
		"postgres_savepoint",
		postgresNamedUniqueMigrationTable,
	)
	require.NoError(t, tableDB.AutoMigrate(&postgresNamedUniqueMigration{}))

	require.NoError(t, tableDB.Transaction(func(savepointTx *gorm.DB) error {
		if err := savepointTx.SavePoint("postgres_migration_test").Error; err != nil {
			return err
		}
		if err := savepointTx.Table(postgresNamedUniqueMigrationTable).Create(
			&postgresNamedUniqueMigration{Key: "rolled-back"},
		).Error; err != nil {
			return err
		}
		return savepointTx.RollbackTo("postgres_migration_test").Error
	}))
	var count int64
	require.NoError(t, tableDB.Model(&postgresNamedUniqueMigration{}).Count(&count).Error)
	assert.Zero(t, count)
}

func TestChooseDBPostgreSQLRetainsErrorTranslation(t *testing.T) {
	db := openPostgresMigrationTestDB(t)
	tableDB, _ := beginPostgresMigrationTestSchema(
		t,
		db,
		"postgres_error_translation",
		postgresNamedUniqueMigrationTable,
	)
	require.NoError(t, tableDB.AutoMigrate(&postgresNamedUniqueMigration{}))
	require.NoError(t, tableDB.Exec(
		"ALTER TABLE ? ADD CONSTRAINT ? UNIQUE (?)",
		clause.Table{Name: postgresNamedUniqueMigrationTable},
		clause.Column{Name: postgresNamedUniqueMigrationConstraint},
		clause.Column{Name: "key"},
	).Error)
	tableDB = tableDB.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)})
	tableDB.Config.TranslateError = true
	require.NoError(t, tableDB.Create(&postgresNamedUniqueMigration{Key: "duplicate"}).Error)
	err := tableDB.Create(&postgresNamedUniqueMigration{Key: "duplicate"}).Error
	assert.ErrorIs(t, err, gorm.ErrDuplicatedKey)
}

func TestChooseDBPostgreSQLRetainsIdentifierLengthConfiguration(t *testing.T) {
	db := openPostgresMigrationTestDB(t)
	namingStrategy, ok := db.Config.NamingStrategy.(schema.NamingStrategy)
	require.True(t, ok)
	assert.Equal(t, 63, namingStrategy.IdentifierMaxLength)
	tableDB, _ := beginPostgresMigrationTestSchema(
		t,
		db,
		"postgres_identifier_length",
		postgresIdentifierLengthMigrationTable,
	)
	require.NoError(t, tableDB.AutoMigrate(&postgresIdentifierLengthMigration{}))
	require.NoError(t, tableDB.AutoMigrate(&postgresIdentifierLengthMigration{}))
	assert.True(t, tableDB.Migrator().HasIndex(
		&postgresIdentifierLengthMigration{},
		"Name",
	))
}
