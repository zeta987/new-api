package model

import (
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// MigrateColumnUnique guards against GORM deriving a default constraint name
// for an existing named PostgreSQL unique constraint or index. The derived
// object may not exist, and attempting to drop it aborts the entire migration.
func (migrator postgresSchemaMigrator) MigrateColumnUnique(
	value any,
	field *schema.Field,
	columnType gorm.ColumnType,
) error {
	unique, ok := columnType.Unique()
	if !ok || field.PrimaryKey {
		return nil
	}

	statement := &gorm.Statement{DB: migrator.DB}
	tableName := ""
	if migrator.DB.Statement != nil {
		tableName = migrator.DB.Statement.Table
	}
	if err := statement.ParseWithSpecialTableName(value, tableName); err != nil {
		return err
	}
	constraintName := migrator.DB.NamingStrategy.UniqueName(statement.Table, field.DBName)

	if unique && !field.Unique {
		if !migrator.HasConstraint(value, constraintName) {
			return nil
		}
		return migrator.DropConstraint(value, constraintName)
	}
	if !unique && field.Unique {
		return migrator.CreateConstraint(value, constraintName)
	}
	return nil
}
