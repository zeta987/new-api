package model

import (
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type postgresMigrationDialector struct {
	gorm.Dialector
}

func (dialector postgresMigrationDialector) Apply(config *gorm.Config) error {
	if applier, ok := dialector.Dialector.(interface {
		Apply(*gorm.Config) error
	}); ok {
		return applier.Apply(config)
	}
	return nil
}

func (dialector postgresMigrationDialector) Migrator(db *gorm.DB) gorm.Migrator {
	return postgresMigrationMigrator{
		Migrator: dialector.Dialector.Migrator(db),
		db:       db,
	}
}

func (dialector postgresMigrationDialector) SavePoint(tx *gorm.DB, name string) error {
	if savePointer, ok := dialector.Dialector.(gorm.SavePointerDialectorInterface); ok {
		return savePointer.SavePoint(tx, name)
	}
	return gorm.ErrUnsupportedDriver
}

func (dialector postgresMigrationDialector) RollbackTo(tx *gorm.DB, name string) error {
	if savePointer, ok := dialector.Dialector.(gorm.SavePointerDialectorInterface); ok {
		return savePointer.RollbackTo(tx, name)
	}
	return gorm.ErrUnsupportedDriver
}

func (dialector postgresMigrationDialector) Translate(err error) error {
	if translator, ok := dialector.Dialector.(gorm.ErrorTranslator); ok {
		return translator.Translate(err)
	}
	return err
}

type postgresMigrationMigrator struct {
	gorm.Migrator
	db *gorm.DB
}

// MigrateColumnUnique guards against GORM deriving a default constraint name
// for an existing named PostgreSQL unique constraint or index. The derived
// object may not exist, and attempting to drop it aborts the entire migration.
func (migrator postgresMigrationMigrator) MigrateColumnUnique(
	value any,
	field *schema.Field,
	columnType gorm.ColumnType,
) error {
	unique, ok := columnType.Unique()
	if !ok || field.PrimaryKey {
		return nil
	}

	statement := &gorm.Statement{DB: migrator.db}
	tableName := ""
	if migrator.db.Statement != nil {
		tableName = migrator.db.Statement.Table
	}
	if err := statement.ParseWithSpecialTableName(value, tableName); err != nil {
		return err
	}
	constraintName := migrator.db.NamingStrategy.UniqueName(statement.Table, field.DBName)

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
