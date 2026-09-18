package model

import (
	"os"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestGetChannelTreatsNullablePriorityAsZero(t *testing.T) {
	tests := []struct {
		name         string
		databaseType common.DatabaseType
		open         func(*testing.T) *gorm.DB
	}{
		{
			name:         "sqlite",
			databaseType: common.DatabaseTypeSQLite,
			open: func(t *testing.T) *gorm.DB {
				db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
				require.NoError(t, err)
				return db
			},
		},
		{
			name:         "mysql",
			databaseType: common.DatabaseTypeMySQL,
			open: func(t *testing.T) *gorm.DB {
				dsn := strings.TrimSpace(os.Getenv("TEST_MYSQL_DSN"))
				if dsn == "" {
					t.Skip("TEST_MYSQL_DSN is not configured")
				}
				db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
				require.NoError(t, err)
				return db
			},
		},
		{
			name:         "postgres",
			databaseType: common.DatabaseTypePostgreSQL,
			open: func(t *testing.T) *gorm.DB {
				dsn := strings.TrimSpace(os.Getenv("TEST_POSTGRES_DSN"))
				if dsn == "" {
					t.Skip("TEST_POSTGRES_DSN is not configured")
				}
				db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
				require.NoError(t, err)
				return db
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testGetChannelTreatsNullablePriorityAsZero(t, test.open(t), test.databaseType)
		})
	}
}

func testGetChannelTreatsNullablePriorityAsZero(
	t *testing.T,
	db *gorm.DB,
	databaseType common.DatabaseType,
) {
	t.Helper()
	previousDB := DB
	previousMainDatabaseType := common.MainDatabaseType()
	previousLogDatabaseType := common.LogDatabaseType()
	previousMemoryCacheEnabled := common.MemoryCacheEnabled
	DB = db
	common.SetMainDatabaseType(databaseType)
	initCol()
	t.Cleanup(func() {
		DB = previousDB
		common.SetDatabaseTypes(previousMainDatabaseType, previousLogDatabaseType)
		common.MemoryCacheEnabled = previousMemoryCacheEnabled
		initCol()
		if previousMemoryCacheEnabled {
			InitChannelCache()
		}
	})

	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	require.NoError(t, db.Migrator().DropTable(&Ability{}, &Channel{}))
	require.NoError(t, db.AutoMigrate(&Channel{}, &Ability{}))
	t.Cleanup(func() {
		require.NoError(t, db.Migrator().DropTable(&Ability{}, &Channel{}))
	})

	positivePriority := int64(100)
	zeroPriority := int64(0)
	channels := []Channel{
		{
			Id:       530001,
			Type:     constant.ChannelTypeZhipu_v4,
			Key:      "positive-priority-key",
			Status:   common.ChannelStatusEnabled,
			Name:     "positive-priority",
			Models:   "nullable-priority-model",
			Group:    "default",
			Priority: &positivePriority,
		},
		{
			Id:       530002,
			Type:     constant.ChannelTypeOpenAI,
			Key:      "zero-priority-key",
			Status:   common.ChannelStatusEnabled,
			Name:     "zero-priority",
			Models:   "nullable-priority-model",
			Group:    "default",
			Priority: &zeroPriority,
		},
		{
			Id:       530003,
			Type:     constant.ChannelTypeAnthropic,
			Key:      "nullable-priority-key",
			Status:   common.ChannelStatusEnabled,
			Name:     "nullable-priority",
			Models:   "nullable-priority-model",
			Group:    "default",
			Priority: &zeroPriority,
		},
	}
	for i := range channels {
		require.NoError(t, db.Create(&channels[i]).Error)
		require.NoError(t, db.Create(&Ability{
			Group:     "default",
			Model:     "nullable-priority-model",
			ChannelId: channels[i].Id,
			Enabled:   true,
			Priority:  channels[i].Priority,
		}).Error)
	}
	require.NoError(t, db.Model(&Ability{}).
		Where("channel_id = ?", channels[2].Id).
		UpdateColumn("priority", nil).Error)

	var stored Ability
	require.NoError(t, db.Where("channel_id = ?", channels[2].Id).First(&stored).Error)
	assert.Nil(t, stored.Priority)

	selectionCases := []struct {
		name               string
		allowedChannelType int
		wantChannelID      int
	}{
		{name: "positive", allowedChannelType: constant.ChannelTypeZhipu_v4, wantChannelID: channels[0].Id},
		{name: "explicit_zero", allowedChannelType: constant.ChannelTypeOpenAI, wantChannelID: channels[1].Id},
		{name: "nullable_zero", allowedChannelType: constant.ChannelTypeAnthropic, wantChannelID: channels[2].Id},
	}
	for _, memoryCacheEnabled := range []bool{false, true} {
		modeName := "database"
		if memoryCacheEnabled {
			modeName = "memory_cache"
		}
		t.Run(modeName, func(t *testing.T) {
			common.MemoryCacheEnabled = memoryCacheEnabled
			if memoryCacheEnabled {
				InitChannelCache()
			}
			for _, selectionCase := range selectionCases {
				t.Run(selectionCase.name, func(t *testing.T) {
					selected, err := GetRandomSatisfiedChannel(
						"default",
						"nullable-priority-model",
						0,
						[]dto.ChannelFilter{{
							Kind:                dto.FilterAllowedChannelTypes,
							AllowedChannelTypes: []int{selectionCase.allowedChannelType},
						}},
					)
					require.NoError(t, err)
					require.NotNil(t, selected)
					assert.Equal(t, selectionCase.wantChannelID, selected.Id)
				})
			}
		})
	}
}
