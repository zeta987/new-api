package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGLM52AliasesRequireZhipuV4Channels(t *testing.T) {
	previousMemoryCacheEnabled := common.MemoryCacheEnabled
	t.Cleanup(func() {
		common.MemoryCacheEnabled = previousMemoryCacheEnabled
		clearGLMRoutingTables(t)
	})

	for _, memoryCacheEnabled := range []bool{false, true} {
		name := "database"
		if memoryCacheEnabled {
			name = "memory cache"
		}
		t.Run(name, func(t *testing.T) {
			clearGLMRoutingTables(t)

			v3Priority := int64(100)
			v4Priority := int64(0)
			v3 := &Channel{
				Id:       525201,
				Type:     constant.ChannelTypeZhipu,
				Key:      "v3-base-key",
				Status:   common.ChannelStatusEnabled,
				Name:     "v3-base",
				Models:   "glm-5.2",
				Group:    "default",
				Priority: &v3Priority,
			}
			v4 := &Channel{
				Id:       525202,
				Type:     constant.ChannelTypeZhipu_v4,
				Key:      "v4-base-key",
				Status:   common.ChannelStatusEnabled,
				Name:     "v4-base",
				Models:   "glm-5.2",
				Group:    "default",
				Priority: &v4Priority,
			}
			for _, channel := range []*Channel{v3, v4} {
				require.NoError(t, DB.Create(channel).Error)
				require.NoError(t, channel.AddAbilities(nil))
			}

			common.MemoryCacheEnabled = memoryCacheEnabled
			if memoryCacheEnabled {
				InitChannelCache()
			}

			for _, alias := range []string{"glm-5.2-none", "glm-5.2-high", "glm-5.2-max"} {
				got, err := GetRandomSatisfiedChannel("default", alias, 0, nil)
				require.NoError(t, err)
				require.NotNil(t, got)
				assert.Equal(t, v4.Id, got.Id)
				assert.False(t, IsChannelEnabledForGroupModel("default", alias, v3.Id))
				assert.True(t, IsChannelEnabledForGroupModel("default", alias, v4.Id))
			}

			got, err := GetRandomSatisfiedChannel("default", "glm-5.2", 0, nil)
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, v3.Id, got.Id)
		})
	}
}

func TestGLM52AliasesRejectV3ExactAndBaseOnlyChannels(t *testing.T) {
	previousMemoryCacheEnabled := common.MemoryCacheEnabled
	t.Cleanup(func() {
		common.MemoryCacheEnabled = previousMemoryCacheEnabled
		clearGLMRoutingTables(t)
	})

	for _, memoryCacheEnabled := range []bool{false, true} {
		name := "database"
		if memoryCacheEnabled {
			name = "memory cache"
		}
		t.Run(name, func(t *testing.T) {
			clearGLMRoutingTables(t)

			v3Priority := int64(100)
			v4Priority := int64(0)
			v3Exact := &Channel{
				Id:       525203,
				Type:     constant.ChannelTypeZhipu,
				Key:      "v3-exact-key",
				Status:   common.ChannelStatusEnabled,
				Name:     "v3-exact",
				Models:   "glm-5.2-high",
				Group:    "default",
				Priority: &v3Priority,
			}
			v4Base := &Channel{
				Id:       525204,
				Type:     constant.ChannelTypeZhipu_v4,
				Key:      "v4-base-key",
				Status:   common.ChannelStatusEnabled,
				Name:     "v4-base",
				Models:   "glm-5.2",
				Group:    "default",
				Priority: &v4Priority,
			}
			for _, channel := range []*Channel{v3Exact, v4Base} {
				require.NoError(t, DB.Create(channel).Error)
				require.NoError(t, channel.AddAbilities(nil))
			}

			common.MemoryCacheEnabled = memoryCacheEnabled
			if memoryCacheEnabled {
				InitChannelCache()
			}

			got, err := GetRandomSatisfiedChannel("default", "glm-5.2-high", 0, nil)
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, v4Base.Id, got.Id)
			assert.False(t, IsChannelEnabledForGroupModel("default", "glm-5.2-high", v3Exact.Id))
			assert.True(t, IsChannelEnabledForGroupModel("default", "glm-5.2-high", v4Base.Id))

			clearGLMRoutingTables(t)
			require.NoError(t, DB.Create(v3Exact).Error)
			require.NoError(t, v3Exact.AddAbilities(nil))
			if memoryCacheEnabled {
				InitChannelCache()
			}

			got, err = GetRandomSatisfiedChannel("default", "glm-5.2-high", 0, nil)
			require.NoError(t, err)
			assert.Nil(t, got)
			assert.False(t, IsChannelEnabledForGroupModel("default", "glm-5.2-high", v3Exact.Id))
		})
	}
}

func TestGLM52AliasRetryUsesNextZhipuV4Priority(t *testing.T) {
	previousMemoryCacheEnabled := common.MemoryCacheEnabled
	t.Cleanup(func() {
		common.MemoryCacheEnabled = previousMemoryCacheEnabled
		clearGLMRoutingTables(t)
	})

	for _, memoryCacheEnabled := range []bool{false, true} {
		name := "database"
		if memoryCacheEnabled {
			name = "memory cache"
		}
		t.Run(name, func(t *testing.T) {
			clearGLMRoutingTables(t)

			highPriority := int64(100)
			lowPriority := int64(0)
			highPriorityChannel := &Channel{
				Id:       525205,
				Type:     constant.ChannelTypeZhipu_v4,
				Key:      "v4-high-priority-key",
				Status:   common.ChannelStatusEnabled,
				Name:     "v4-high-priority",
				Models:   "glm-5.2",
				Group:    "default",
				Priority: &highPriority,
			}
			lowPriorityChannel := &Channel{
				Id:       525206,
				Type:     constant.ChannelTypeZhipu_v4,
				Key:      "v4-low-priority-key",
				Status:   common.ChannelStatusEnabled,
				Name:     "v4-low-priority",
				Models:   "glm-5.2",
				Group:    "default",
				Priority: &lowPriority,
			}
			for _, channel := range []*Channel{highPriorityChannel, lowPriorityChannel} {
				require.NoError(t, DB.Create(channel).Error)
				require.NoError(t, channel.AddAbilities(nil))
			}

			common.MemoryCacheEnabled = memoryCacheEnabled
			if memoryCacheEnabled {
				InitChannelCache()
			}

			got, err := GetRandomSatisfiedChannel("default", "glm-5.2-high", 1, nil)
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, lowPriorityChannel.Id, got.Id)
		})
	}
}

func TestGLM52ExactZhipuV4AliasPrecedesBaseCandidate(t *testing.T) {
	previousMemoryCacheEnabled := common.MemoryCacheEnabled
	t.Cleanup(func() {
		common.MemoryCacheEnabled = previousMemoryCacheEnabled
		clearGLMRoutingTables(t)
	})

	for _, memoryCacheEnabled := range []bool{false, true} {
		name := "database"
		if memoryCacheEnabled {
			name = "memory cache"
		}
		t.Run(name, func(t *testing.T) {
			clearGLMRoutingTables(t)

			basePriority := int64(100)
			exactPriority := int64(0)
			baseChannel := &Channel{
				Id:       525207,
				Type:     constant.ChannelTypeZhipu_v4,
				Key:      "v4-base-key",
				Status:   common.ChannelStatusEnabled,
				Name:     "v4-base",
				Models:   "glm-5.2",
				Group:    "default",
				Priority: &basePriority,
			}
			exactChannel := &Channel{
				Id:       525208,
				Type:     constant.ChannelTypeZhipu_v4,
				Key:      "v4-exact-key",
				Status:   common.ChannelStatusEnabled,
				Name:     "v4-exact",
				Models:   "glm-5.2-high",
				Group:    "default",
				Priority: &exactPriority,
			}
			for _, channel := range []*Channel{baseChannel, exactChannel} {
				require.NoError(t, DB.Create(channel).Error)
				require.NoError(t, channel.AddAbilities(nil))
			}

			common.MemoryCacheEnabled = memoryCacheEnabled
			if memoryCacheEnabled {
				InitChannelCache()
			}

			got, err := GetRandomSatisfiedChannel("default", "glm-5.2-high", 0, nil)
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, exactChannel.Id, got.Id)
		})
	}
}

func clearGLMRoutingTables(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.Exec("DELETE FROM abilities").Error)
	require.NoError(t, DB.Exec("DELETE FROM channels").Error)
}
