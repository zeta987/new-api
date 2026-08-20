package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// GLM effort aliases are served by every channel type whose adaptor translates
// the alias into an upstream reasoning effort, and by no other type.
func TestGLMEffortAliasesReachOpenAICompatibleChannels(t *testing.T) {
	previousMemoryCacheEnabled := common.MemoryCacheEnabled
	t.Cleanup(func() {
		common.MemoryCacheEnabled = previousMemoryCacheEnabled
		clearGLMRoutingTables(t)
	})

	cases := []struct {
		name        string
		channelType int
		alias       string
		baseModel   string
		wantServed  bool
	}{
		{name: "openai serves glm-5.3", channelType: constant.ChannelTypeOpenAI, alias: "glm-5.3-high", baseModel: "glm-5.3", wantServed: true},
		{name: "openrouter serves glm-5.3", channelType: constant.ChannelTypeOpenRouter, alias: "glm-5.3-max", baseModel: "glm-5.3", wantServed: true},
		{name: "openai serves glm-5.2", channelType: constant.ChannelTypeOpenAI, alias: "glm-5.2-none", baseModel: "glm-5.2", wantServed: true},
		{name: "openrouter serves glm-5.2", channelType: constant.ChannelTypeOpenRouter, alias: "glm-5.2-high", baseModel: "glm-5.2", wantServed: true},
		{name: "zhipu v4 still serves", channelType: constant.ChannelTypeZhipu_v4, alias: "glm-5.3-low", baseModel: "glm-5.3", wantServed: true},
		{name: "zhipu v3 stays excluded", channelType: constant.ChannelTypeZhipu, alias: "glm-5.3-high", baseModel: "glm-5.3", wantServed: false},
		{name: "anthropic stays excluded", channelType: constant.ChannelTypeAnthropic, alias: "glm-5.2-max", baseModel: "glm-5.2", wantServed: false},
	}

	for _, memoryCacheEnabled := range []bool{false, true} {
		selectionPath := "database"
		if memoryCacheEnabled {
			selectionPath = "memory cache"
		}
		t.Run(selectionPath, func(t *testing.T) {
			for _, testCase := range cases {
				t.Run(testCase.name, func(t *testing.T) {
					clearGLMRoutingTables(t)

					priority := int64(0)
					channel := &Channel{
						Id:       610001,
						Type:     testCase.channelType,
						Key:      "alias-key",
						Status:   common.ChannelStatusEnabled,
						Name:     "alias-channel",
						Models:   testCase.baseModel,
						Group:    "default",
						Priority: &priority,
					}
					require.NoError(t, DB.Create(channel).Error)
					require.NoError(t, channel.AddAbilities(nil))

					common.MemoryCacheEnabled = memoryCacheEnabled
					if memoryCacheEnabled {
						InitChannelCache()
					}

					got, err := GetRandomSatisfiedChannel("default", testCase.alias, 0, "")
					if testCase.wantServed {
						require.NoError(t, err)
						require.NotNil(t, got)
						assert.Equal(t, channel.Id, got.Id)
						assert.True(t, IsChannelEnabledForGroupModel("default", testCase.alias, channel.Id))
						return
					}
					assert.True(t, err != nil || got == nil, "alias must not select channel type %d", testCase.channelType)
					assert.False(t, IsChannelEnabledForGroupModel("default", testCase.alias, channel.Id))
				})
			}
		})
	}
}
