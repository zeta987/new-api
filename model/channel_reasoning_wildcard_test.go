package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func clearReasoningWildcardTables(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.Exec("DELETE FROM abilities").Error)
	require.NoError(t, DB.Exec("DELETE FROM channels").Error)
}

func TestChannelModelCandidatesIncludeGPT56ReasoningWildcard(t *testing.T) {
	assert.Equal(t, []string{
		"gpt-5.6-luna-pro-max",
		"gpt-5.6-luna-*",
		"gpt-5.6-luna",
	}, ModelMatchCandidates("gpt-5.6-luna-pro-max"))

	assert.Equal(t, []string{
		"gpt-5.6-luna-pro-ultra",
	}, ModelMatchCandidates("gpt-5.6-luna-pro-ultra"))

	assert.Empty(t, ModelMatchCandidates("gpt-5.6-luna-*"))
}

func TestGetRandomSatisfiedChannelPrefersGPT56WildcardOverBase(t *testing.T) {
	previousMemoryCacheEnabled := common.MemoryCacheEnabled
	t.Cleanup(func() {
		common.MemoryCacheEnabled = previousMemoryCacheEnabled
		clearReasoningWildcardTables(t)
	})

	for _, memoryCacheEnabled := range []bool{false, true} {
		name := "database"
		if memoryCacheEnabled {
			name = "memory cache"
		}
		t.Run(name, func(t *testing.T) {
			clearReasoningWildcardTables(t)

			priority := int64(0)
			channels := []*Channel{
				{
					Id:       5604,
					Type:     1,
					Key:      "exact-key",
					Status:   common.ChannelStatusEnabled,
					Name:     "gpt56-exact",
					Models:   "gpt-5.6-luna-pro-high",
					Group:    "default",
					Priority: &priority,
				},
				{
					Id:       5602,
					Type:     1,
					Key:      "base-key",
					Status:   common.ChannelStatusEnabled,
					Name:     "gpt56-base",
					Models:   "gpt-5.6-luna",
					Group:    "default",
					Priority: &priority,
				},
				{
					Id:       5603,
					Type:     1,
					Key:      "wildcard-key",
					Status:   common.ChannelStatusEnabled,
					Name:     "gpt56-wildcard",
					Models:   "gpt-5.6-luna-*",
					Group:    "default",
					Priority: &priority,
				},
			}
			for _, channel := range channels {
				require.NoError(t, DB.Create(channel).Error)
				require.NoError(t, channel.AddAbilities(nil))
			}

			common.MemoryCacheEnabled = memoryCacheEnabled
			if memoryCacheEnabled {
				InitChannelCache()
			}

			got, err := GetRandomSatisfiedChannel("default", "gpt-5.6-luna-pro-max", 0, "")
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, 5603, got.Id)

			got, err = GetRandomSatisfiedChannel("default", "gpt-5.6-luna-pro-high", 0, "")
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, 5604, got.Id)
		})
	}
}

func TestGetRandomSatisfiedChannelFallsBackAfterRequestPathFilter(t *testing.T) {
	previousMemoryCacheEnabled := common.MemoryCacheEnabled
	t.Cleanup(func() {
		common.MemoryCacheEnabled = previousMemoryCacheEnabled
		clearReasoningWildcardTables(t)
	})

	for _, memoryCacheEnabled := range []bool{false, true} {
		name := "database"
		if memoryCacheEnabled {
			name = "memory cache"
		}
		t.Run(name, func(t *testing.T) {
			clearReasoningWildcardTables(t)

			priority := int64(0)
			channels := []*Channel{
				{
					Id:       5605,
					Type:     constant.ChannelTypeAdvancedCustom,
					Key:      "exact-key",
					Status:   common.ChannelStatusEnabled,
					Name:     "gpt56-exact-unsupported-path",
					Models:   "gpt-5.6-luna-pro-max",
					Group:    "default",
					Priority: &priority,
				},
				{
					Id:       5606,
					Type:     constant.ChannelTypeOpenAI,
					Key:      "wildcard-key",
					Status:   common.ChannelStatusEnabled,
					Name:     "gpt56-wildcard",
					Models:   "gpt-5.6-luna-*",
					Group:    "default",
					Priority: &priority,
				},
			}
			for _, channel := range channels {
				require.NoError(t, DB.Create(channel).Error)
				require.NoError(t, channel.AddAbilities(nil))
			}

			common.MemoryCacheEnabled = memoryCacheEnabled
			if memoryCacheEnabled {
				InitChannelCache()
			}

			got, err := GetRandomSatisfiedChannel("default", "gpt-5.6-luna-pro-max", 0, "/v1/responses")
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, 5606, got.Id)
		})
	}
}

func TestGetRandomSatisfiedChannelKeepsExactCandidateAfterPriorityPathFilter(t *testing.T) {
	previousMemoryCacheEnabled := common.MemoryCacheEnabled
	t.Cleanup(func() {
		common.MemoryCacheEnabled = previousMemoryCacheEnabled
		clearReasoningWildcardTables(t)
	})

	for _, memoryCacheEnabled := range []bool{false, true} {
		name := "database"
		if memoryCacheEnabled {
			name = "memory cache"
		}
		t.Run(name, func(t *testing.T) {
			clearReasoningWildcardTables(t)

			highPriority := int64(10)
			mediumPriority := int64(5)
			lowPriority := int64(0)
			wildcardPriority := int64(100)
			channels := []*Channel{
				{
					Id:       5607,
					Type:     constant.ChannelTypeAdvancedCustom,
					Key:      "exact-high-key",
					Status:   common.ChannelStatusEnabled,
					Name:     "gpt56-exact-high-unsupported-path",
					Models:   "gpt-5.6-luna-pro-max",
					Group:    "default",
					Priority: &highPriority,
				},
				{
					Id:       5608,
					Type:     constant.ChannelTypeOpenAI,
					Key:      "exact-medium-key",
					Status:   common.ChannelStatusEnabled,
					Name:     "gpt56-exact-medium",
					Models:   "gpt-5.6-luna-pro-max",
					Group:    "default",
					Priority: &mediumPriority,
				},
				{
					Id:       5610,
					Type:     constant.ChannelTypeOpenAI,
					Key:      "exact-low-key",
					Status:   common.ChannelStatusEnabled,
					Name:     "gpt56-exact-low",
					Models:   "gpt-5.6-luna-pro-max",
					Group:    "default",
					Priority: &lowPriority,
				},
				{
					Id:       5609,
					Type:     constant.ChannelTypeOpenAI,
					Key:      "wildcard-key",
					Status:   common.ChannelStatusEnabled,
					Name:     "gpt56-wildcard-high",
					Models:   "gpt-5.6-luna-*",
					Group:    "default",
					Priority: &wildcardPriority,
				},
			}
			for _, channel := range channels {
				require.NoError(t, DB.Create(channel).Error)
				require.NoError(t, channel.AddAbilities(nil))
			}

			common.MemoryCacheEnabled = memoryCacheEnabled
			if memoryCacheEnabled {
				InitChannelCache()
			}

			got, err := GetRandomSatisfiedChannel("default", "gpt-5.6-luna-pro-max", 0, "/v1/responses")
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, 5608, got.Id)

			got, err = GetRandomSatisfiedChannel("default", "gpt-5.6-luna-pro-max", 1, "/v1/responses")
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, 5610, got.Id)
		})
	}
}

func TestGetRandomSatisfiedChannelMatchesGPT56ReasoningWildcard(t *testing.T) {
	previousMemoryCacheEnabled := common.MemoryCacheEnabled
	t.Cleanup(func() {
		common.MemoryCacheEnabled = previousMemoryCacheEnabled
		clearReasoningWildcardTables(t)
	})

	for _, memoryCacheEnabled := range []bool{false, true} {
		name := "database"
		if memoryCacheEnabled {
			name = "memory cache"
		}
		t.Run(name, func(t *testing.T) {
			clearReasoningWildcardTables(t)

			priority := int64(0)
			channel := &Channel{
				Id:       5601,
				Type:     1,
				Key:      "test-key",
				Status:   common.ChannelStatusEnabled,
				Name:     "gpt56-wildcard",
				Models:   "gpt-5.6-luna-*",
				Group:    "default",
				Priority: &priority,
			}
			require.NoError(t, DB.Create(channel).Error)
			require.NoError(t, channel.AddAbilities(nil))

			common.MemoryCacheEnabled = memoryCacheEnabled
			if memoryCacheEnabled {
				InitChannelCache()
			}

			got, err := GetRandomSatisfiedChannel("default", "gpt-5.6-luna-pro-max", 0, "")
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, channel.Id, got.Id)
			assert.True(t, IsChannelEnabledForGroupModel("default", "gpt-5.6-luna-pro-max", channel.Id))

			got, err = GetRandomSatisfiedChannel("default", "gpt-5.6-luna-pro-ultra", 0, "")
			require.NoError(t, err)
			assert.Nil(t, got)
			assert.False(t, IsChannelEnabledForGroupModel("default", "gpt-5.6-luna-pro-ultra", channel.Id))

			got, err = GetRandomSatisfiedChannel("default", "gpt-5.6-luna-*", 0, "")
			require.NoError(t, err)
			assert.Nil(t, got)
			assert.False(t, IsChannelEnabledForGroupModel("default", "gpt-5.6-luna-*", channel.Id))
		})
	}
}
