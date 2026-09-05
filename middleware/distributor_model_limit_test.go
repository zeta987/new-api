package middleware

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/stretchr/testify/assert"
)

func TestTokenAllowsGPT56ReasoningModelCandidates(t *testing.T) {
	tests := []struct {
		name         string
		modelLimits  map[string]bool
		requestModel string
		want         bool
	}{
		{
			name:         "exact model",
			modelLimits:  map[string]bool{"gpt-5.6-luna-pro-max": true},
			requestModel: "gpt-5.6-luna-pro-max",
			want:         true,
		},
		{
			name:         "reasoning wildcard",
			modelLimits:  map[string]bool{"gpt-5.6-luna-*": true},
			requestModel: "gpt-5.6-luna-pro-max",
			want:         true,
		},
		{
			name:         "normalized base",
			modelLimits:  map[string]bool{"gpt-5.6-luna": true},
			requestModel: "gpt-5.6-luna-pro-max",
			want:         true,
		},
		{
			name:         "invalid suffix cannot use wildcard",
			modelLimits:  map[string]bool{"gpt-5.6-luna-*": true},
			requestModel: "gpt-5.6-luna-pro-ultra",
			want:         false,
		},
		{
			name:         "literal wildcard is configuration only",
			modelLimits:  map[string]bool{"gpt-5.6-luna-*": true},
			requestModel: "gpt-5.6-luna-*",
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tokenAllowsModel(tt.modelLimits, tt.requestModel))
		})
	}
}

func TestGLMReasoningAllowedChannelTypesFollowRelayFormat(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		model string
		want  []int
		ok    bool
	}{
		{
			name:  "chat completions",
			path:  "/v1/chat/completions",
			model: "glm-5.3-flash-high",
			want:  []int{constant.ChannelTypeZhipu_v4, constant.ChannelTypeOpenAI, constant.ChannelTypeOpenRouter},
			ok:    true,
		},
		{
			name:  "playground chat completions",
			path:  "/pg/chat/completions",
			model: "glm-5.3-flash-low",
			want:  []int{constant.ChannelTypeZhipu_v4, constant.ChannelTypeOpenAI, constant.ChannelTypeOpenRouter},
			ok:    true,
		},
		{
			name:  "responses only allows zhipu v4",
			path:  "/v1/responses",
			model: "glm-5.3-flash-max",
			want:  []int{constant.ChannelTypeZhipu_v4},
			ok:    true,
		},
		{name: "claude format rejects alias", path: "/v1/messages", model: "glm-5.3-flash-high", want: []int{}, ok: true},
		{name: "gemini format rejects alias", path: "/v1beta/models/glm-5.3-flash-high:generateContent", model: "glm-5.3-flash-high", want: []int{}, ok: true},
		{name: "bare glm has no suffix policy", path: "/v1/chat/completions", model: "glm-5.3-flash"},
		{name: "malformed glm alias has no suffix policy", path: "/v1/chat/completions", model: "glm-low"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got, ok := reasoningAllowedChannelTypes(testCase.path, testCase.model)
			assert.Equal(t, testCase.ok, ok)
			assert.Equal(t, testCase.want, got)
		})
	}
}

func TestQwenChatAliasChannelPolicy(t *testing.T) {
	settings := model_setting.GetGlobalSettings()
	previous := settings.ThinkingModelBlacklist
	t.Cleanup(func() { settings.ThinkingModelBlacklist = previous })
	settings.ThinkingModelBlacklist = append(append([]string(nil), previous...), "qwen3.9-max-low@temperature:0.2")
	for _, tc := range []struct {
		model string
		path  string
		want  bool
	}{
		{"qwen3.8-max-low", "/v1/chat/completions", true},
		{"qwen3.8-flash-none@temperature:0.2", "/pg/chat/completions", true},
		{"qwen3.9-max-low@temperature:0.2", "/v1/chat/completions", false},
		{"qwen3.8-max", "/v1/chat/completions", false},
		{"qwen3.8-max-low", "/v1/responses", false},
		{"qwen3.8-max-low", "/v1/messages", false},
	} {
		t.Run(tc.model+tc.path, func(t *testing.T) {
			allowed, restricted := reasoningAllowedChannelTypes(tc.path, tc.model)
			assert.Equal(t, tc.want, restricted)
			if tc.want {
				assert.Equal(t, []int{constant.ChannelTypeOpenAI}, allowed)
			} else {
				assert.Nil(t, allowed)
			}
		})
	}
}
