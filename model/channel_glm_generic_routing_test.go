package model

import (
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExplicitGLMAllowedChannelTypesReachThirdPartyChatAbilities(t *testing.T) {
	previousMemoryCacheEnabled := common.MemoryCacheEnabled
	t.Cleanup(func() {
		common.MemoryCacheEnabled = previousMemoryCacheEnabled
		clearGLMRoutingTables(t)
	})

	allowedTypes := []int{
		constant.ChannelTypeZhipu_v4,
		constant.ChannelTypeOpenAI,
		constant.ChannelTypeOpenRouter,
	}
	for _, memoryCacheEnabled := range []bool{false, true} {
		for _, channelType := range []int{constant.ChannelTypeOpenAI, constant.ChannelTypeOpenRouter} {
			name := fmt.Sprintf("type %d/database", channelType)
			if memoryCacheEnabled {
				name = fmt.Sprintf("type %d/memory cache", channelType)
			}
			t.Run(name, func(t *testing.T) {
				clearGLMRoutingTables(t)
				priority := int64(0)
				channel := &Channel{
					Id:       535300 + channelType,
					Type:     channelType,
					Key:      "third-party-key",
					Status:   common.ChannelStatusEnabled,
					Name:     "third-party",
					Models:   "glm-5.3-flash",
					Group:    "default",
					Priority: &priority,
				}
				require.NoError(t, DB.Create(channel).Error)
				require.NoError(t, channel.AddAbilities(nil))

				common.MemoryCacheEnabled = memoryCacheEnabled
				if memoryCacheEnabled {
					InitChannelCache()
				}

				filters := []dto.ChannelFilter{{
					Kind:                dto.FilterAllowedChannelTypes,
					AllowedChannelTypes: allowedTypes,
				}}
				got, err := GetRandomSatisfiedChannel("default", "glm-5.3-flash-high", 0, filters)
				require.NoError(t, err)
				require.NotNil(t, got)
				assert.Equal(t, channel.Id, got.Id)
				assert.True(t, IsChannelEnabledForGroupModel("default", "glm-5.3-flash-high", channel.Id))
			})
		}
	}
}

func TestExplicitGLMAllowedChannelTypesRejectZhipuV3(t *testing.T) {
	previousMemoryCacheEnabled := common.MemoryCacheEnabled
	t.Cleanup(func() {
		common.MemoryCacheEnabled = previousMemoryCacheEnabled
		clearGLMRoutingTables(t)
	})

	priority := int64(0)
	channel := &Channel{
		Id:       535399,
		Type:     constant.ChannelTypeZhipu,
		Key:      "zhipu-v3-key",
		Status:   common.ChannelStatusEnabled,
		Name:     "zhipu-v3",
		Models:   "glm-5.3-flash",
		Group:    "default",
		Priority: &priority,
	}
	require.NoError(t, DB.Create(channel).Error)
	require.NoError(t, channel.AddAbilities(nil))

	filters := []dto.ChannelFilter{{
		Kind: dto.FilterAllowedChannelTypes,
		AllowedChannelTypes: []int{
			constant.ChannelTypeZhipu_v4,
			constant.ChannelTypeOpenAI,
			constant.ChannelTypeOpenRouter,
		},
	}}
	got, err := GetRandomSatisfiedChannel("default", "glm-5.3-flash-high", 0, filters)
	require.NoError(t, err)
	assert.Nil(t, got)
}
