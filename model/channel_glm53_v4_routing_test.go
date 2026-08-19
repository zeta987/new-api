package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGLM53AliasesRequireZhipuV4Channels(t *testing.T) {
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
				Id:       525301,
				Type:     constant.ChannelTypeZhipu,
				Key:      "v3-base-key",
				Status:   common.ChannelStatusEnabled,
				Name:     "v3-base",
				Models:   "glm-5.3",
				Group:    "default",
				Priority: &v3Priority,
			}
			v4 := &Channel{
				Id:       525302,
				Type:     constant.ChannelTypeZhipu_v4,
				Key:      "v4-base-key",
				Status:   common.ChannelStatusEnabled,
				Name:     "v4-base",
				Models:   "glm-5.3",
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

			for _, alias := range []string{"glm-5.3-low", "glm-5.3-high", "glm-5.3-max"} {
				got, err := GetRandomSatisfiedChannel("default", alias, 0, "")
				require.NoError(t, err)
				require.NotNil(t, got)
				assert.Equal(t, v4.Id, got.Id)
				assert.False(t, IsChannelEnabledForGroupModel("default", alias, v3.Id))
				assert.True(t, IsChannelEnabledForGroupModel("default", alias, v4.Id))
			}

			got, err := GetRandomSatisfiedChannel("default", "glm-5.3", 0, "")
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, v3.Id, got.Id)
		})
	}
}
