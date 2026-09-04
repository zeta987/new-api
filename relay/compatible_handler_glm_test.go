package relay

import (
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/stretchr/testify/assert"
)

func TestResolveChatRequestHandlingForcesGLMAliasesThroughChatAdaptor(t *testing.T) {
	tests := []struct {
		name               string
		model              string
		globalPass         bool
		channelPass        bool
		responsesBridge    bool
		wantResponses      bool
		wantRawPassThrough bool
	}{
		{name: "glm alias bypasses responses bridge", model: "glm-5.3-flash-high", responsesBridge: true},
		{name: "glm alias bypasses global pass through", model: "glm-5.3-flash-high", globalPass: true},
		{name: "glm alias bypasses channel pass through", model: "glm-5.3-flash-high", channelPass: true},
		{name: "ordinary model uses responses bridge", model: "gpt-4.1", responsesBridge: true, wantResponses: true},
		{name: "ordinary model uses global pass through", model: "gpt-4.1", globalPass: true, wantRawPassThrough: true},
		{name: "bare glm keeps configured pass through", model: "glm-5.3-flash", channelPass: true, wantRawPassThrough: true},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			info := &relaycommon.RelayInfo{
				OriginModelName: testCase.model,
				RelayMode:       relayconstant.RelayModeChatCompletions,
				ChannelMeta: &relaycommon.ChannelMeta{
					ChannelSetting: dto.ChannelSettings{PassThroughBodyEnabled: testCase.channelPass},
				},
			}

			useResponses, useRawPassThrough := resolveChatRequestHandling(info, testCase.globalPass, testCase.responsesBridge)

			assert.Equal(t, testCase.wantResponses, useResponses)
			assert.Equal(t, testCase.wantRawPassThrough, useRawPassThrough)
		})
	}
}
