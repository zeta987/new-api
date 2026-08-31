package model

import (
	"os"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestChooseDBPostgreSQLRepeatedAutoMigrateNamedUniqueIndex(t *testing.T) {
	db := openPostgresMigrationTestDB(t)
	require.NoError(t, db.Migrator().DropTable(postgresNamedUniqueMigrationTable))
	t.Cleanup(func() {
		require.NoError(t, db.Migrator().DropTable(postgresNamedUniqueMigrationTable))
	})

	tableDB := db.Table(postgresNamedUniqueMigrationTable)
	require.NoError(t, tableDB.AutoMigrate(&postgresNamedUniqueMigration{}))
	require.NoError(t, db.Exec(
		"ALTER TABLE ? ADD CONSTRAINT ? UNIQUE (?)",
		clause.Table{Name: postgresNamedUniqueMigrationTable},
		clause.Column{Name: postgresNamedUniqueMigrationConstraint},
		clause.Column{Name: "key"},
	).Error)
	require.NoError(t, tableDB.AutoMigrate(&postgresNamedUniqueMigration{}))
	assert.True(t, tableDB.Migrator().HasConstraint(
		&postgresNamedUniqueMigration{},
		postgresNamedUniqueMigrationConstraint,
	))
	assert.True(t, tableDB.Migrator().HasIndex(
		&postgresNamedUniqueMigration{},
		postgresNamedUniqueMigrationIndex,
	))
}

func TestChooseDBPostgreSQLRetainsSavepointSupport(t *testing.T) {
	db := openPostgresMigrationTestDB(t)
	require.NoError(t, db.Migrator().DropTable(postgresNamedUniqueMigrationTable))
	t.Cleanup(func() {
		require.NoError(t, db.Migrator().DropTable(postgresNamedUniqueMigrationTable))
	})
	tableDB := db.Table(postgresNamedUniqueMigrationTable)
	require.NoError(t, tableDB.AutoMigrate(&postgresNamedUniqueMigration{}))

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		if err := tx.SavePoint("postgres_migration_test").Error; err != nil {
			return err
		}
		if err := tx.Table(postgresNamedUniqueMigrationTable).Create(
			&postgresNamedUniqueMigration{Key: "rolled-back"},
		).Error; err != nil {
			return err
		}
		return tx.RollbackTo("postgres_migration_test").Error
	}))
	var count int64
	require.NoError(t, tableDB.Model(&postgresNamedUniqueMigration{}).Count(&count).Error)
	assert.Zero(t, count)
}

func TestChooseDBPostgreSQLRetainsErrorTranslation(t *testing.T) {
	db := openPostgresMigrationTestDB(t)
	require.NoError(t, db.Migrator().DropTable(postgresNamedUniqueMigrationTable))
	t.Cleanup(func() {
		require.NoError(t, db.Migrator().DropTable(postgresNamedUniqueMigrationTable))
	})

	tableDB := db.Table(postgresNamedUniqueMigrationTable)
	require.NoError(t, tableDB.AutoMigrate(&postgresNamedUniqueMigration{}))
	require.NoError(t, db.Exec(
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
	require.NoError(t, db.Migrator().DropTable(postgresIdentifierLengthMigrationTable))
	t.Cleanup(func() {
		require.NoError(t, db.Migrator().DropTable(postgresIdentifierLengthMigrationTable))
	})

	tableDB := db.Table(postgresIdentifierLengthMigrationTable)
	require.NoError(t, tableDB.AutoMigrate(&postgresIdentifierLengthMigration{}))
	require.NoError(t, tableDB.AutoMigrate(&postgresIdentifierLengthMigration{}))
	assert.True(t, tableDB.Migrator().HasIndex(
		&postgresIdentifierLengthMigration{},
		"Name",
	))
}
