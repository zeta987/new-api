package zhipu_4v_test

import (
	"encoding/json"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relay/channel/zhipu"
	"github.com/QuantumNous/new-api/relay/channel/zhipu_4v"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGLM52ReasoningEffortAliasesOverrideBody(t *testing.T) {
	for _, effort := range []string{"none", "high", "max"} {
		t.Run(effort, func(t *testing.T) {
			alias := "glm-5.2-" + effort
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
			got := requireGLM52Request(t, converted)

			assert.Equal(t, "glm-5.2", got.Model)
			assert.Equal(t, effort, got.ReasoningEffort)
			assert.Equal(t, "glm-5.2", info.UpstreamModelName)
			assert.Equal(t, effort, info.ReasoningEffort)
			assert.JSONEq(t,
				`{"model":"glm-5.2","messages":[{"role":"user","content":"hi"}],"reasoning_effort":"`+effort+`","thinking":{"type":"enabled"},"stop":null}`,
				marshalRequest(t, got),
			)
		})
	}
}

func TestGLM52AliasConvertsWithoutChannelMeta(t *testing.T) {
	request := &dto.GeneralOpenAIRequest{Model: "glm-5.2-max"}
	info := &relaycommon.RelayInfo{OriginModelName: "glm-5.2-max"}

	require.NotPanics(t, func() {
		converted, err := (&zhipu_4v.Adaptor{}).ConvertOpenAIRequest(nil, info, request)
		require.NoError(t, err)
		got := requireGLM52Request(t, converted)
		assert.Equal(t, "glm-5.2", got.Model)
		assert.Equal(t, "max", got.ReasoningEffort)
		assert.Equal(t, "max", info.ReasoningEffort)
		assert.JSONEq(t, `{"model":"glm-5.2","reasoning_effort":"max","stop":null}`, marshalRequest(t, got))
	})
}

func TestGLM52BareModelPreservesExplicitReasoningEffort(t *testing.T) {
	request := &dto.GeneralOpenAIRequest{Model: "glm-5.2", ReasoningEffort: "high"}
	info := &relaycommon.RelayInfo{
		OriginModelName: "glm-5.2",
		ChannelMeta:     &relaycommon.ChannelMeta{UpstreamModelName: "glm-5.2"},
	}

	converted, err := (&zhipu_4v.Adaptor{}).ConvertOpenAIRequest(nil, info, request)
	require.NoError(t, err)
	got := requireGLM52Request(t, converted)

	assert.Equal(t, "glm-5.2", got.Model)
	assert.Equal(t, "high", got.ReasoningEffort)
	assert.Equal(t, "glm-5.2", info.UpstreamModelName)
	assert.Equal(t, "high", info.ReasoningEffort)
	assert.JSONEq(t, `{"model":"glm-5.2","reasoning_effort":"high","stop":null}`, marshalRequest(t, got))
}

func TestGLM52BareModelOmitsEmptyReasoningEffort(t *testing.T) {
	request := &dto.GeneralOpenAIRequest{Model: "glm-5.2"}
	info := &relaycommon.RelayInfo{
		OriginModelName: "glm-5.2",
		ChannelMeta:     &relaycommon.ChannelMeta{UpstreamModelName: "glm-5.2"},
	}

	converted, err := (&zhipu_4v.Adaptor{}).ConvertOpenAIRequest(nil, info, request)
	require.NoError(t, err)
	got := requireGLM52Request(t, converted)

	assert.Equal(t, "glm-5.2", got.Model)
	assert.Empty(t, got.ReasoningEffort)
	assert.Equal(t, "glm-5.2", info.UpstreamModelName)
	assert.Empty(t, info.ReasoningEffort)
	assert.JSONEq(t, `{"model":"glm-5.2","stop":null}`, marshalRequest(t, got))
}

func TestGLM52RecoversOriginAliasOnlyForBaseMappedUpstreamModel(t *testing.T) {
	t.Run("base upstream model recovers origin alias", func(t *testing.T) {
		request := &dto.GeneralOpenAIRequest{Model: "glm-5.2"}
		info := &relaycommon.RelayInfo{
			OriginModelName: "glm-5.2-max",
			ChannelMeta:     &relaycommon.ChannelMeta{UpstreamModelName: "glm-5.2"},
		}

		converted, err := (&zhipu_4v.Adaptor{}).ConvertOpenAIRequest(nil, info, request)
		require.NoError(t, err)
		got := requireGLM52Request(t, converted)

		assert.Equal(t, "glm-5.2", got.Model)
		assert.Equal(t, "max", got.ReasoningEffort)
		assert.Equal(t, "glm-5.2", info.UpstreamModelName)
		assert.Equal(t, "max", info.ReasoningEffort)
		assert.JSONEq(t, `{"model":"glm-5.2","reasoning_effort":"max","stop":null}`, marshalRequest(t, got))
	})

	t.Run("other mapped upstream model does not recover origin alias", func(t *testing.T) {
		request := &dto.GeneralOpenAIRequest{Model: "custom-glm-5.2"}
		info := &relaycommon.RelayInfo{
			OriginModelName: "glm-5.2-max",
			ChannelMeta:     &relaycommon.ChannelMeta{UpstreamModelName: "custom-glm-5.2"},
		}

		converted, err := (&zhipu_4v.Adaptor{}).ConvertOpenAIRequest(nil, info, request)
		require.NoError(t, err)
		got := requireGLM52Request(t, converted)

		assert.Equal(t, "custom-glm-5.2", got.Model)
		assert.Empty(t, got.ReasoningEffort)
		assert.Equal(t, "custom-glm-5.2", info.UpstreamModelName)
		assert.Empty(t, info.ReasoningEffort)
		assert.JSONEq(t, `{"model":"custom-glm-5.2","stop":null}`, marshalRequest(t, got))
	})
}

func TestGLM52InvalidAliasesRemainUnchanged(t *testing.T) {
	for _, model := range []string{"glm-5.2-low", "glm-5.2-xhigh", "glm-5.2-max-extra", "glm-5.1-high", "GLM-5.2-high"} {
		t.Run(model, func(t *testing.T) {
			request := &dto.GeneralOpenAIRequest{Model: model}
			info := &relaycommon.RelayInfo{
				OriginModelName: model,
				ChannelMeta:     &relaycommon.ChannelMeta{UpstreamModelName: model},
			}

			converted, err := (&zhipu_4v.Adaptor{}).ConvertOpenAIRequest(nil, info, request)
			require.NoError(t, err)
			got := requireGLM52Request(t, converted)

			assert.Equal(t, model, got.Model)
			assert.Empty(t, got.ReasoningEffort)
			assert.Equal(t, model, info.UpstreamModelName)
			assert.Empty(t, info.ReasoningEffort)
			assert.JSONEq(t, `{"model":"`+model+`","stop":null}`, marshalRequest(t, got))
		})
	}
}

func TestZhipu4VModelListIncludesGLM52AliasesOnly(t *testing.T) {
	modelList := (&zhipu_4v.Adaptor{}).GetModelList()
	for _, model := range []string{"glm-5.2", "glm-5.2-none", "glm-5.2-high", "glm-5.2-max"} {
		assert.Contains(t, modelList, model)
	}
	for _, model := range []string{"glm-5.2-low", "glm-5.2-xhigh", "glm-5.2-max-extra"} {
		assert.NotContains(t, modelList, model)
	}
}

func TestZhipuV3ModelListExcludesGLM52Models(t *testing.T) {
	modelList := (&zhipu.Adaptor{}).GetModelList()
	for _, model := range []string{"glm-5.2", "glm-5.2-none", "glm-5.2-high", "glm-5.2-max"} {
		assert.NotContains(t, modelList, model)
	}
}

func requireGLM52Request(t *testing.T, converted any) *dto.GeneralOpenAIRequest {
	t.Helper()
	require.IsType(t, &dto.GeneralOpenAIRequest{}, converted)
	request, ok := converted.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	return request
}

func marshalRequest(t *testing.T, request *dto.GeneralOpenAIRequest) string {
	t.Helper()
	payload, err := common.Marshal(request)
	require.NoError(t, err)
	return string(payload)
}
