package zhipu_4v_test

import (
	"encoding/json"
	"testing"

	"github.com/QuantumNous/new-api/relay/channel/zhipu"
	"github.com/QuantumNous/new-api/relay/channel/zhipu_4v"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGLM53ReasoningEffortAliasesOverrideBody(t *testing.T) {
	for _, effort := range []string{"low", "high", "max"} {
		t.Run(effort, func(t *testing.T) {
			alias := "glm-5.3-" + effort
			bodyEffort := "high"
			if effort == bodyEffort {
				bodyEffort = "max"
			}
			request := &dto.GeneralOpenAIRequest{
				Model:           alias,
				Messages:        []dto.Message{{Role: "user", Content: "hi"}},
				ReasoningEffort: bodyEffort,
				THINKING:        json.RawMessage(`{"type":"enabled"}`),
			}
			info := &relaycommon.RelayInfo{
				OriginModelName: alias,
				ChannelMeta:     &relaycommon.ChannelMeta{UpstreamModelName: alias},
			}

			converted, err := (&zhipu_4v.Adaptor{}).ConvertOpenAIRequest(nil, info, request)
			require.NoError(t, err)
			got := requireZhipuRequest(t, converted)

			assert.Equal(t, "glm-5.3", got.Model)
			assert.Equal(t, effort, got.ReasoningEffort)
			assert.Equal(t, "glm-5.3", info.UpstreamModelName)
			assert.Equal(t, effort, info.ReasoningEffort)
			assert.JSONEq(t,
				`{"model":"glm-5.3","messages":[{"role":"user","content":"hi"}],"reasoning_effort":"`+effort+`","thinking":{"type":"enabled"},"stop":null}`,
				marshalRequest(t, got),
			)
		})
	}
}

func TestGLM53BareModelPreservesExplicitReasoningEffort(t *testing.T) {
	request := &dto.GeneralOpenAIRequest{Model: "glm-5.3", ReasoningEffort: "low"}
	info := &relaycommon.RelayInfo{
		OriginModelName: "glm-5.3",
		ChannelMeta:     &relaycommon.ChannelMeta{UpstreamModelName: "glm-5.3"},
	}

	converted, err := (&zhipu_4v.Adaptor{}).ConvertOpenAIRequest(nil, info, request)
	require.NoError(t, err)
	got := requireZhipuRequest(t, converted)

	assert.Equal(t, "glm-5.3", got.Model)
	assert.Equal(t, "low", got.ReasoningEffort)
	assert.Equal(t, "low", info.ReasoningEffort)
	assert.JSONEq(t, `{"model":"glm-5.3","reasoning_effort":"low","stop":null}`, marshalRequest(t, got))
}

func TestGLM53BareModelOmitsEmptyReasoningEffort(t *testing.T) {
	request := &dto.GeneralOpenAIRequest{Model: "glm-5.3"}
	info := &relaycommon.RelayInfo{
		OriginModelName: "glm-5.3",
		ChannelMeta:     &relaycommon.ChannelMeta{UpstreamModelName: "glm-5.3"},
	}

	converted, err := (&zhipu_4v.Adaptor{}).ConvertOpenAIRequest(nil, info, request)
	require.NoError(t, err)
	got := requireZhipuRequest(t, converted)

	assert.Equal(t, "glm-5.3", got.Model)
	assert.Empty(t, got.ReasoningEffort)
	assert.Empty(t, info.ReasoningEffort)
	assert.JSONEq(t, `{"model":"glm-5.3","stop":null}`, marshalRequest(t, got))
}

// A channel may map a GLM-5.2 alias onto GLM-5.3. Recovering the origin alias
// across base models would send an effort the target model never advertised.
func TestGLMOriginAliasRecoveryRequiresMatchingBaseModel(t *testing.T) {
	t.Run("same base model recovers the origin alias", func(t *testing.T) {
		request := &dto.GeneralOpenAIRequest{Model: "glm-5.3"}
		info := &relaycommon.RelayInfo{
			OriginModelName: "glm-5.3-low",
			ChannelMeta:     &relaycommon.ChannelMeta{UpstreamModelName: "glm-5.3"},
		}

		converted, err := (&zhipu_4v.Adaptor{}).ConvertOpenAIRequest(nil, info, request)
		require.NoError(t, err)
		got := requireZhipuRequest(t, converted)

		assert.Equal(t, "glm-5.3", got.Model)
		assert.Equal(t, "low", got.ReasoningEffort)
		assert.Equal(t, "low", info.ReasoningEffort)
	})

	t.Run("other base model does not recover the origin alias", func(t *testing.T) {
		request := &dto.GeneralOpenAIRequest{Model: "glm-5.3"}
		info := &relaycommon.RelayInfo{
			OriginModelName: "glm-5.2-none",
			ChannelMeta:     &relaycommon.ChannelMeta{UpstreamModelName: "glm-5.3"},
		}

		converted, err := (&zhipu_4v.Adaptor{}).ConvertOpenAIRequest(nil, info, request)
		require.NoError(t, err)
		got := requireZhipuRequest(t, converted)

		assert.Equal(t, "glm-5.3", got.Model)
		assert.Empty(t, got.ReasoningEffort)
		assert.Empty(t, info.ReasoningEffort)
		assert.JSONEq(t, `{"model":"glm-5.3","stop":null}`, marshalRequest(t, got))
	})
}

func TestGLM53InvalidAliasesRemainUnchanged(t *testing.T) {
	for _, model := range []string{"glm-low", "glm--low", "GLM-5.3-high", "custom-glm-5.3-high"} {
		t.Run(model, func(t *testing.T) {
			request := &dto.GeneralOpenAIRequest{Model: model, ReasoningEffort: "high"}
			info := &relaycommon.RelayInfo{
				OriginModelName: model,
				ChannelMeta:     &relaycommon.ChannelMeta{UpstreamModelName: model},
			}

			converted, err := (&zhipu_4v.Adaptor{}).ConvertOpenAIRequest(nil, info, request)
			require.NoError(t, err)
			got := requireZhipuRequest(t, converted)

			assert.Equal(t, model, got.Model)
			assert.Empty(t, got.ReasoningEffort)
			assert.Empty(t, info.ReasoningEffort)
			assert.JSONEq(t, `{"model":"`+model+`","stop":null}`, marshalRequest(t, got))
		})
	}
}

func TestZhipu4VModelListIncludesGLM53FlashAliases(t *testing.T) {
	modelList := (&zhipu_4v.Adaptor{}).GetModelList()
	for _, model := range []string{"glm-5.3", "glm-5.3-low", "glm-5.3-high", "glm-5.3-max", "glm-5.3-flash", "glm-5.3-flash-low", "glm-5.3-flash-high", "glm-5.3-flash-max"} {
		assert.Contains(t, modelList, model)
	}
	for _, model := range []string{"glm-5.3-max-extra", "glm-5.3-flash-fast"} {
		assert.NotContains(t, modelList, model)
	}
}

func TestZhipuV3ModelListExcludesGLM53Models(t *testing.T) {
	modelList := (&zhipu.Adaptor{}).GetModelList()
	for _, model := range []string{"glm-5.3", "glm-5.3-low", "glm-5.3-high", "glm-5.3-max", "glm-5.3-flash", "glm-5.3-flash-low", "glm-5.3-flash-high", "glm-5.3-flash-max"} {
		assert.NotContains(t, modelList, model)
	}
}
