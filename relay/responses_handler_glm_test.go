package relay

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/stretchr/testify/assert"
)

func TestShouldUseRawResponsesPassThroughForGLMAliases(t *testing.T) {
	tests := []struct {
		name        string
		channelType int
		model       string
		globalPass  bool
		channelPass bool
		want        bool
	}{
		{name: "zhipu alias bypasses global pass through", channelType: constant.ChannelTypeZhipu_v4, model: "glm-5.3-flash-high", globalPass: true},
		{name: "zhipu alias bypasses channel pass through", channelType: constant.ChannelTypeZhipu_v4, model: "glm-5.3-flash-high", channelPass: true},
		{name: "zhipu bare model keeps pass through", channelType: constant.ChannelTypeZhipu_v4, model: "glm-5.3-flash", globalPass: true, want: true},
		{name: "other channel keeps pass through", channelType: constant.ChannelTypeOpenAI, model: "glm-5.3-flash-high", globalPass: true, want: true},
		{name: "disabled pass through stays disabled", channelType: constant.ChannelTypeZhipu_v4, model: "glm-5.3-flash-high"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			info := &relaycommon.RelayInfo{
				OriginModelName: testCase.model,
				RelayMode:       relayconstant.RelayModeResponses,
				ChannelMeta: &relaycommon.ChannelMeta{
					ChannelType:    testCase.channelType,
					ChannelSetting: dto.ChannelSettings{PassThroughBodyEnabled: testCase.channelPass},
				},
			}

			assert.Equal(t, testCase.want, shouldUseRawResponsesPassThrough(info, testCase.globalPass))
		})
	}
}
