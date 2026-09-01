package model

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func testPrefillGroupMigrationNonPostgreSQL(t *testing.T, db *gorm.DB) {
	t.Helper()
	tableName := fmt.Sprintf("prefill_group_migration_%d", time.Now().UnixNano())
	t.Cleanup(func() { _ = db.Migrator().DropTable(tableName) })

	tableDB := db.Table(tableName)
	require.NoError(t, tableDB.AutoMigrate(&PrefillGroup{}))
	require.NoError(t, tableDB.Create(&PrefillGroup{
		Name:        "preserved-name",
		Type:        "model",
		Items:       JSONValue(`["gpt-test"]`),
		Description: "preserve me",
	}).Error)

	for range 2 {
		require.NoError(t, migratePrefillGroupUniqueness(db))
		require.NoError(t, tableDB.AutoMigrate(&PrefillGroup{}))
	}

	var preserved PrefillGroup
	require.NoError(t, tableDB.Where("name = ?", "preserved-name").First(&preserved).Error)
	assert.Equal(t, "preserve me", preserved.Description)
	assert.True(t, tableDB.Migrator().HasIndex(&PrefillGroup{}, prefillGroupNameIndex))
}

func TestMigratePrefillGroupUniquenessSQLite(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	testPrefillGroupMigrationNonPostgreSQL(t, db)
}

func TestMigratePrefillGroupUniquenessMySQL(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("TEST_MYSQL_DSN"))
	if dsn == "" {
		t.Skip("TEST_MYSQL_DSN is not configured")
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	testPrefillGroupMigrationNonPostgreSQL(t, db)
}

type prefillMigrationExpectation struct {
	legacyConstraintCount int64
	legacyIndexCount      int64
	deletedNameReusable   bool
}

func TestMigratePrefillGroupUniquenessPostgreSQL(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("TEST_POSTGRES_DSN"))
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN is not configured")
	}

	db, err := gorm.Open(postgresMigrationDialector{Dialector: postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true,
	})}, &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })

	t.Run("invalid_legacy_index_from_failed_concurrent_build_is_atomic", func(t *testing.T) {
		schemaName := fmt.Sprintf("prefill_group_invalid_legacy_%d", time.Now().UnixNano())
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
		require.NoError(t, setupTx.Exec(`
CREATE TABLE prefill_groups (
    id bigserial PRIMARY KEY,
    name varchar(64) NOT NULL
)`).Error)
		require.NoError(t, setupTx.Exec(
			"INSERT INTO prefill_groups (name) VALUES (?), (?)",
			"duplicate-name",
			"duplicate-name",
		).Error)
		require.NoError(t, setupTx.Commit().Error)

		concurrentIndexErr := db.Exec(
			"CREATE UNIQUE INDEX CONCURRENTLY ? ON ? (?)",
			clause.Column{Name: legacyPrefillGroupNameUnique},
			clause.Table{Name: schemaName + ".prefill_groups"},
			clause.Column{Name: "name"},
		).Error
		require.Error(t, concurrentIndexErr)

		fixtureTx := db.Begin()
		require.NoError(t, fixtureTx.Error)
		t.Cleanup(func() { _ = fixtureTx.Rollback().Error })
		require.NoError(t, fixtureTx.Exec(
			"SET LOCAL search_path TO ?",
			clause.Table{Name: schemaName},
		).Error)
		require.NoError(t, fixtureTx.Exec(`
DELETE FROM prefill_groups
WHERE id = (SELECT max(id) FROM prefill_groups)`).Error)
		require.NoError(t, fixtureTx.Commit().Error)

		migrationTx := db.Begin()
		require.NoError(t, migrationTx.Error)
		t.Cleanup(func() { _ = migrationTx.Rollback().Error })
		require.NoError(t, migrationTx.Exec(
			"SET LOCAL search_path TO ?",
			clause.Table{Name: schemaName},
		).Error)

		type legacyIndexSnapshot struct {
			Valid      bool   `gorm:"column:index_valid"`
			Ready      bool   `gorm:"column:index_ready"`
			Definition string `gorm:"column:index_definition"`
		}
		inspectLegacyIndex := func() legacyIndexSnapshot {
			t.Helper()
			var snapshot legacyIndexSnapshot
			require.NoError(t, migrationTx.Raw(`
SELECT index_meta.indisvalid AS index_valid,
       index_meta.indisready AS index_ready,
       pg_get_indexdef(index_meta.indexrelid) AS index_definition
FROM pg_catalog.pg_index AS index_meta
JOIN pg_catalog.pg_class AS index_class
  ON index_class.oid = index_meta.indexrelid
WHERE index_meta.indrelid = to_regclass('prefill_groups')
  AND index_class.relname = ?`, legacyPrefillGroupNameUnique).Scan(&snapshot).Error)
			return snapshot
		}
		type prefillRow struct {
			ID   int64
			Name string
		}
		inspectRows := func() []prefillRow {
			t.Helper()
			var rows []prefillRow
			require.NoError(t, migrationTx.Raw(
				"SELECT id, name FROM prefill_groups ORDER BY id",
			).Scan(&rows).Error)
			return rows
		}

		legacyBefore := inspectLegacyIndex()
		rowsBefore := inspectRows()
		require.False(t, legacyBefore.Valid)
		require.False(t, legacyBefore.Ready)
		require.NotEmpty(t, legacyBefore.Definition)
		require.False(t, migrationTx.Migrator().HasColumn(&PrefillGroup{}, "DeletedAt"))
		require.False(t, migrationTx.Migrator().HasIndex(&PrefillGroup{}, prefillGroupNameIndex))

		migrationErr := migratePrefillGroupUniqueness(migrationTx)
		require.Error(t, migrationErr)
		assert.Contains(t, migrationErr.Error(), legacyPrefillGroupNameUnique)
		assert.Contains(t, migrationErr.Error(), "unexpected definition")
		assert.Equal(t, legacyBefore, inspectLegacyIndex())
		assert.Equal(t, rowsBefore, inspectRows())
		assert.False(t, migrationTx.Migrator().HasColumn(&PrefillGroup{}, "DeletedAt"))
		assert.False(t, migrationTx.Migrator().HasIndex(&PrefillGroup{}, prefillGroupNameIndex))
	})

	t.Run("malformed_target_without_legacy_is_atomic", func(t *testing.T) {
		tx := db.Begin()
		require.NoError(t, tx.Error)
		t.Cleanup(func() { _ = tx.Rollback().Error })

		schemaName := fmt.Sprintf("prefill_group_invalid_target_%d", time.Now().UnixNano())
		require.NoError(t, tx.Exec(
			"CREATE SCHEMA ?",
			clause.Table{Name: schemaName},
		).Error)
		require.NoError(t, tx.Exec(
			"SET LOCAL search_path TO ?",
			clause.Table{Name: schemaName},
		).Error)
		require.NoError(t, tx.Exec(`
CREATE TABLE prefill_groups (
    id bigserial PRIMARY KEY,
    name varchar(64) NOT NULL
)`).Error)
		require.NoError(t, tx.Exec(
			"CREATE INDEX ? ON ? (?)",
			clause.Column{Name: prefillGroupNameIndex},
			clause.Table{Name: "prefill_groups"},
			clause.Column{Name: "name"},
		).Error)

		err := migratePrefillGroupUniqueness(tx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected definition")
		assert.False(t, tx.Migrator().HasColumn(&PrefillGroup{}, "DeletedAt"))
		assert.True(t, tx.Migrator().HasIndex(&PrefillGroup{}, prefillGroupNameIndex))
	})

	t.Run("malformed_target_after_preflight_is_rejected_before_mutation", func(t *testing.T) {
		schemaName := fmt.Sprintf("prefill_group_interleaving_%d", time.Now().UnixNano())
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
		require.NoError(t, setupTx.Exec(`
CREATE TABLE prefill_groups (
    id bigserial PRIMARY KEY,
    name varchar(64) NOT NULL,
    CONSTRAINT idx_prefill_groups_name UNIQUE (name)
)`).Error)
		require.NoError(t, setupTx.Commit().Error)

		type migrationContextKey struct{}
		migrationMarker := &struct{}{}
		lockReached := make(chan struct{})
		resumeMigration := make(chan struct{})
		var releaseOnce sync.Once
		defer releaseOnce.Do(func() { close(resumeMigration) })
		schemaMutationAttempted := false
		callbackName := "test:prefill_bridge_interleaving"
		require.NoError(t, db.Callback().Raw().Before("gorm:raw").Register(
			callbackName,
			func(callbackDB *gorm.DB) {
				if callbackDB.Statement.Context.Value(migrationContextKey{}) != migrationMarker {
					return
				}
				sql := callbackDB.Statement.SQL.String()
				if strings.HasPrefix(sql, "ALTER TABLE") {
					schemaMutationAttempted = true
				}
				if sql != `LOCK TABLE "prefill_groups" IN ACCESS EXCLUSIVE MODE` {
					return
				}
				select {
				case lockReached <- struct{}{}:
				case <-callbackDB.Statement.Context.Done():
					return
				}
				select {
				case <-resumeMigration:
				case <-callbackDB.Statement.Context.Done():
				}
			},
		))

		ctx, cancel := context.WithTimeout(
			context.WithValue(context.Background(), migrationContextKey{}, migrationMarker),
			8*time.Second,
		)
		defer cancel()
		migrationTx := db.WithContext(ctx).Begin()
		require.NoError(t, migrationTx.Error)
		t.Cleanup(func() { _ = migrationTx.Rollback().Error })
		require.NoError(t, migrationTx.Exec(
			"SET LOCAL search_path TO ?",
			clause.Table{Name: schemaName},
		).Error)
		migrationResult := make(chan error, 1)
		go func() {
			migrationResult <- migratePrefillGroupUniqueness(migrationTx)
		}()

		select {
		case <-lockReached:
		case <-ctx.Done():
			require.NoError(t, ctx.Err())
		}
		malformedTx := db.Begin()
		require.NoError(t, malformedTx.Error)
		t.Cleanup(func() { _ = malformedTx.Rollback().Error })
		require.NoError(t, malformedTx.Exec(
			"SET LOCAL search_path TO ?",
			clause.Table{Name: schemaName},
		).Error)
		malformedErr := malformedTx.Exec(
			"CREATE INDEX ? ON ? (?)",
			clause.Column{Name: prefillGroupNameIndex},
			clause.Table{Name: "prefill_groups"},
			clause.Column{Name: "name"},
		).Error
		if malformedErr == nil {
			malformedErr = malformedTx.Commit().Error
		}
		releaseOnce.Do(func() { close(resumeMigration) })
		require.NoError(t, malformedErr)

		migrationErr := <-migrationResult
		require.Error(t, migrationErr)
		assert.Contains(t, migrationErr.Error(), "unexpected definition")
		assert.False(t, schemaMutationAttempted)
		assert.False(t, migrationTx.Migrator().HasColumn(&PrefillGroup{}, "DeletedAt"))
		targetIndex, err := inspectPrefillGroupNameIndex(migrationTx, "prefill_groups")
		require.NoError(t, err)
		assert.True(t, targetIndex.exists)
		assert.False(t, targetIndex.valid)
	})

	tests := []struct {
		name               string
		prepareOld         func(*testing.T, *gorm.DB)
		blockedConstraints []string
		blockedIndexes     []string
		preservedIndexes   []string
		expectation        prefillMigrationExpectation
	}{
		{
			name: "fresh",
			expectation: prefillMigrationExpectation{
				legacyConstraintCount: 0,
				legacyIndexCount:      0,
				deletedNameReusable:   true,
			},
		},
		{
			name: "legacy_constraint",
			prepareOld: func(t *testing.T, tx *gorm.DB) {
				t.Helper()
				require.NoError(t, tx.Exec(
					"ALTER TABLE ? ADD CONSTRAINT ? UNIQUE (?)",
					clause.Table{Name: "prefill_groups"},
					clause.Column{Name: legacyPrefillGroupNameUnique},
					clause.Column{Name: "name"},
				).Error)
			},
			expectation: prefillMigrationExpectation{
				legacyConstraintCount: 1,
				legacyIndexCount:      0,
				deletedNameReusable:   false,
			},
		},
		{
			name: "legacy_standalone_index",
			prepareOld: func(t *testing.T, tx *gorm.DB) {
				t.Helper()
				require.NoError(t, tx.Migrator().DropIndex(&PrefillGroup{}, prefillGroupNameIndex))
				require.NoError(t, tx.Exec(
					"CREATE UNIQUE INDEX ? ON ? (?)",
					clause.Column{Name: legacyPrefillGroupNameUnique},
					clause.Table{Name: "prefill_groups"},
					clause.Column{Name: "name"},
				).Error)
			},
			expectation: prefillMigrationExpectation{
				legacyConstraintCount: 0,
				legacyIndexCount:      1,
				deletedNameReusable:   false,
			},
		},
		{
			name: "arbitrary_constraint_name",
			prepareOld: func(t *testing.T, tx *gorm.DB) {
				t.Helper()
				for _, constraintName := range []string{
					legacyPrefillGroupNameUnique,
					"prefill_groups_name_key",
				} {
					require.NoError(t, tx.Exec(
						"ALTER TABLE ? ADD CONSTRAINT ? UNIQUE (?)",
						clause.Table{Name: "prefill_groups"},
						clause.Column{Name: constraintName},
						clause.Column{Name: "name"},
					).Error)
				}
			},
			blockedConstraints: []string{legacyPrefillGroupNameUnique, "prefill_groups_name_key"},
		},
		{
			name: "arbitrary_index_name",
			prepareOld: func(t *testing.T, tx *gorm.DB) {
				t.Helper()
				for _, indexName := range []string{
					legacyPrefillGroupNameUnique,
					"prefill_groups_name_key",
				} {
					require.NoError(t, tx.Exec(
						"CREATE UNIQUE INDEX ? ON ? (?)",
						clause.Column{Name: indexName},
						clause.Table{Name: "prefill_groups"},
						clause.Column{Name: "name"},
					).Error)
				}
			},
			blockedIndexes: []string{legacyPrefillGroupNameUnique, "prefill_groups_name_key"},
		},
		{
			name: "non_conflicting_indexes_are_preserved",
			prepareOld: func(t *testing.T, tx *gorm.DB) {
				t.Helper()
				require.NoError(t, tx.Exec(
					"ALTER TABLE ? ADD CONSTRAINT ? UNIQUE (?)",
					clause.Table{Name: "prefill_groups"},
					clause.Column{Name: legacyPrefillGroupNameUnique},
					clause.Column{Name: "name"},
				).Error)
				require.NoError(t, tx.Exec(
					"CREATE UNIQUE INDEX ? ON ? (?, ?)",
					clause.Column{Name: "keep_prefill_name_deleted_at"},
					clause.Table{Name: "prefill_groups"},
					clause.Column{Name: "name"},
					clause.Column{Name: "deleted_at"},
				).Error)
				require.NoError(t, tx.Exec(
					"CREATE UNIQUE INDEX ? ON ? (lower(?)) WHERE deleted_at IS NULL",
					clause.Column{Name: "keep_prefill_lower_name"},
					clause.Table{Name: "prefill_groups"},
					clause.Column{Name: "name"},
				).Error)
				require.NoError(t, tx.Exec(
					"CREATE UNIQUE INDEX ? ON ? (?) WHERE deleted_at IS NOT NULL",
					clause.Column{Name: "keep_prefill_deleted_name"},
					clause.Table{Name: "prefill_groups"},
					clause.Column{Name: "name"},
				).Error)
			},
			preservedIndexes: []string{
				"keep_prefill_name_deleted_at",
				"keep_prefill_lower_name",
				"keep_prefill_deleted_name",
			},
			expectation: prefillMigrationExpectation{
				legacyConstraintCount: 1,
				legacyIndexCount:      0,
				deletedNameReusable:   false,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tx := db.Begin()
			require.NoError(t, tx.Error)
			t.Cleanup(func() { _ = tx.Rollback().Error })

			schemaName := fmt.Sprintf("prefill_group_migration_%d", time.Now().UnixNano())
			require.NoError(t, tx.Exec(
				"CREATE SCHEMA ?",
				clause.Table{Name: schemaName},
			).Error)
			require.NoError(t, tx.Exec(
				"SET LOCAL search_path TO ?",
				clause.Table{Name: schemaName},
			).Error)

			require.NoError(t, migratePrefillGroupUniqueness(tx))
			require.NoError(t, tx.AutoMigrate(&PrefillGroup{}))
			original := PrefillGroup{
				Name:        "shared-name",
				Type:        "model",
				Items:       JSONValue(`["gpt-test"]`),
				Description: "preserve me",
			}
			require.NoError(t, tx.Create(&original).Error)
			if test.prepareOld != nil {
				test.prepareOld(t, tx)
			}
			if len(test.blockedConstraints) > 0 || len(test.blockedIndexes) > 0 {
				err := migratePrefillGroupUniqueness(tx)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "prefill_groups_name_key")
				for _, constraintName := range test.blockedConstraints {
					assert.True(t, tx.Migrator().HasConstraint(&PrefillGroup{}, constraintName))
				}
				for _, indexName := range test.blockedIndexes {
					assert.True(t, tx.Migrator().HasIndex(&PrefillGroup{}, indexName))
				}
				return
			}

			for range 2 {
				require.NoError(t, migratePrefillGroupUniqueness(tx))
				require.NoError(t, tx.AutoMigrate(&PrefillGroup{}))
			}
			for _, indexName := range test.preservedIndexes {
				assert.True(t, tx.Migrator().HasIndex(&PrefillGroup{}, indexName))
			}

			var preserved PrefillGroup
			require.NoError(t, tx.First(&preserved, original.Id).Error)
			assert.Equal(t, original.Name, preserved.Name)
			assert.Equal(t, original.Description, preserved.Description)

			var globalConstraintCount int64
			require.NoError(t, tx.Raw(`
SELECT count(*)
FROM pg_catalog.pg_constraint AS constraint_meta
WHERE constraint_meta.conrelid = to_regclass('prefill_groups')
  AND constraint_meta.contype = 'u'
  AND cardinality(constraint_meta.conkey) = 1
  AND EXISTS (
      SELECT 1
      FROM pg_catalog.pg_attribute AS attribute_meta
      WHERE attribute_meta.attrelid = constraint_meta.conrelid
        AND attribute_meta.attnum = constraint_meta.conkey[1]
        AND attribute_meta.attname = 'name'
  )`).Scan(&globalConstraintCount).Error)
			assert.Equal(t, test.expectation.legacyConstraintCount, globalConstraintCount)

			var globalIndexCount int64
			require.NoError(t, tx.Raw(`
SELECT count(*)
FROM pg_catalog.pg_index AS index_meta
JOIN pg_catalog.pg_attribute AS attribute_meta
  ON attribute_meta.attrelid = index_meta.indrelid
 AND attribute_meta.attnum = index_meta.indkey[0]
WHERE index_meta.indrelid = to_regclass('prefill_groups')
  AND index_meta.indisunique
  AND NOT index_meta.indisprimary
  AND index_meta.indpred IS NULL
  AND index_meta.indexprs IS NULL
  AND index_meta.indnatts = 1
  AND attribute_meta.attname = 'name'
  AND NOT EXISTS (
      SELECT 1
      FROM pg_catalog.pg_constraint AS constraint_meta
      WHERE constraint_meta.conindid = index_meta.indexrelid
  )`).Scan(&globalIndexCount).Error)
			assert.Equal(t, test.expectation.legacyIndexCount, globalIndexCount)

			var targetIndexDefinition string
			require.NoError(t, tx.Raw(`
SELECT indexdef
FROM pg_catalog.pg_indexes
WHERE schemaname = current_schema()
  AND tablename = 'prefill_groups'
  AND indexname = ?`, prefillGroupNameIndex).Scan(&targetIndexDefinition).Error)
			assert.Contains(t, strings.ToLower(targetIndexDefinition), "unique index")
			assert.Contains(t, strings.ToLower(targetIndexDefinition), "where (deleted_at is null)")

			duplicateError := tx.Transaction(func(duplicateTx *gorm.DB) error {
				return duplicateTx.Create(&PrefillGroup{
					Name:  original.Name,
					Type:  "model",
					Items: JSONValue(`[]`),
				}).Error
			})
			require.Error(t, duplicateError)

			require.NoError(t, tx.Delete(&original).Error)
			reuseError := tx.Transaction(func(reuseTx *gorm.DB) error {
				return reuseTx.Create(&PrefillGroup{
					Name:  original.Name,
					Type:  "model",
					Items: JSONValue(`[]`),
				}).Error
			})
			if test.expectation.deletedNameReusable {
				require.NoError(t, reuseError)
			} else {
				require.Error(t, reuseError)
			}

			var totalRows int64
			require.NoError(t, tx.Unscoped().Model(&PrefillGroup{}).Count(&totalRows).Error)
			if test.expectation.deletedNameReusable {
				assert.EqualValues(t, 2, totalRows)
			} else {
				assert.EqualValues(t, 1, totalRows)
			}
		})
	}
}
