package deepseek

import (
	"fmt"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestDeepSeekModelListIncludesCurrentLowSuffixes(t *testing.T) {
	models := (&Adaptor{}).GetModelList()

	assert.Contains(t, models, "deepseek-flash-low")
	assert.Contains(t, models, "deepseek-v4-pro-low")
}

func TestApplyDeepSeekV4LowSuffixAcrossRelayFormats(t *testing.T) {
	t.Run("OpenAI chat completions", func(t *testing.T) {
		request := &dto.GeneralOpenAIRequest{Model: "deepseek-v4-pro-low"}
		info := deepSeekLowTestRelayInfo("deepseek-v4-pro-low")

		err := applyDeepSeekV4OpenAIThinkingSuffix(info, request)

		require.NoError(t, err)
		assert.Equal(t, "deepseek-v4-pro", request.Model)
		assert.Equal(t, "enabled", gjson.GetBytes(request.THINKING, "type").String())
		assert.Equal(t, "low", request.ReasoningEffort)
		assert.Equal(t, "deepseek-v4-pro", info.UpstreamModelName)
		assert.Equal(t, "low", info.ReasoningEffort)
	})

	t.Run("Anthropic messages", func(t *testing.T) {
		request := &dto.ClaudeRequest{Model: "deepseek-v4-pro-low"}
		info := deepSeekLowTestRelayInfo("deepseek-v4-pro-low")

		err := applyDeepSeekV4ClaudeThinkingSuffix(info, request)

		require.NoError(t, err)
		assert.Equal(t, "deepseek-v4-pro", request.Model)
		require.NotNil(t, request.Thinking)
		assert.Equal(t, "enabled", request.Thinking.Type)
		assert.Equal(t, "low", gjson.GetBytes(request.OutputConfig, "effort").String())
		assert.Equal(t, "deepseek-v4-pro", info.UpstreamModelName)
		assert.Equal(t, "low", info.ReasoningEffort)
	})

	t.Run("Responses", func(t *testing.T) {
		request := &dto.OpenAIResponsesRequest{Model: "deepseek-v4-pro-low"}
		info := deepSeekLowTestRelayInfo("deepseek-v4-pro-low")

		applyDeepSeekV4ResponsesThinkingSuffix(info, request)

		assert.Equal(t, "deepseek-v4-pro", request.Model)
		require.NotNil(t, request.Reasoning)
		assert.Equal(t, "low", request.Reasoning.Effort)
		assert.Equal(t, "deepseek-v4-pro", info.UpstreamModelName)
		assert.Equal(t, "low", info.ReasoningEffort)
	})
}

// deepSeekLowTestRelayInfo builds a relay info whose upstream name defaults to
// the origin name. Pass upstream to model what a model mapping leaves behind.
func deepSeekLowTestRelayInfo(model string, upstream ...string) *relaycommon.RelayInfo {
	upstreamModel := model
	if len(upstream) > 0 {
		upstreamModel = upstream[0]
	}
	return &relaycommon.RelayInfo{
		OriginModelName: model,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: upstreamModel,
		},
	}
}

// A model mapping that replaces the suffixed alias with the base model (or a
// pinned snapshot of it) leaves the effort only on the origin name.
func TestApplyDeepSeekV4SuffixRecoversEffortAfterMapping(t *testing.T) {
	t.Run("OpenAI chat completions", func(t *testing.T) {
		request := &dto.GeneralOpenAIRequest{Model: "deepseek-v4-pro"}
		info := deepSeekLowTestRelayInfo("deepseek-v4-pro-low", "deepseek-v4-pro")

		require.NoError(t, applyDeepSeekV4OpenAIThinkingSuffix(info, request))
		assert.Equal(t, "deepseek-v4-pro", request.Model)
		assert.Equal(t, "enabled", gjson.GetBytes(request.THINKING, "type").String())
		assert.Equal(t, "low", request.ReasoningEffort)
		assert.Equal(t, "low", info.ReasoningEffort)
	})

	t.Run("Anthropic messages", func(t *testing.T) {
		request := &dto.ClaudeRequest{Model: "deepseek-v4-pro"}
		info := deepSeekLowTestRelayInfo("deepseek-v4-pro-low", "deepseek-v4-pro")

		require.NoError(t, applyDeepSeekV4ClaudeThinkingSuffix(info, request))
		assert.Equal(t, "deepseek-v4-pro", request.Model)
		require.NotNil(t, request.Thinking)
		assert.Equal(t, "enabled", request.Thinking.Type)
		assert.Equal(t, "low", gjson.GetBytes(request.OutputConfig, "effort").String())
		assert.Equal(t, "low", info.ReasoningEffort)
	})

	t.Run("Responses", func(t *testing.T) {
		request := &dto.OpenAIResponsesRequest{Model: "deepseek-v4-pro"}
		info := deepSeekLowTestRelayInfo("deepseek-v4-pro-low", "deepseek-v4-pro")

		applyDeepSeekV4ResponsesThinkingSuffix(info, request)
		assert.Equal(t, "deepseek-v4-pro", request.Model)
		require.NotNil(t, request.Reasoning)
		assert.Equal(t, "low", request.Reasoning.Effort)
		assert.Equal(t, "low", info.ReasoningEffort)
	})

	t.Run("snapshot mapping target is preserved", func(t *testing.T) {
		request := &dto.GeneralOpenAIRequest{Model: "deepseek-v4-pro-0501"}
		info := deepSeekLowTestRelayInfo("deepseek-v4-pro-high", "deepseek-v4-pro-0501")

		require.NoError(t, applyDeepSeekV4OpenAIThinkingSuffix(info, request))
		assert.Equal(t, "deepseek-v4-pro-0501", request.Model)
		assert.Equal(t, "high", request.ReasoningEffort)
		assert.Equal(t, "deepseek-v4-pro-0501", info.UpstreamModelName)
		assert.Equal(t, "high", info.ReasoningEffort)
	})

	t.Run("unrelated mapping target is left alone", func(t *testing.T) {
		request := &dto.GeneralOpenAIRequest{Model: "deepseek-chat"}
		info := deepSeekLowTestRelayInfo("deepseek-v4-pro-high", "deepseek-chat")

		require.NoError(t, applyDeepSeekV4OpenAIThinkingSuffix(info, request))
		assert.Equal(t, "deepseek-chat", request.Model)
		assert.Empty(t, request.ReasoningEffort)
		assert.Empty(t, info.ReasoningEffort)
	})
}

func TestApplyDeepSeekV4HighSuffixAcrossRelayFormats(t *testing.T) {
	for _, base := range []string{"deepseek-flash", "deepseek-v4-pro"} {
		t.Run(base, func(t *testing.T) {
			alias := base + "-high"
			assert.Contains(t, (&Adaptor{}).GetModelList(), alias)

			info := deepSeekLowTestRelayInfo(alias)
			chat := &dto.GeneralOpenAIRequest{Model: alias, ReasoningEffort: "max"}
			require.NoError(t, applyDeepSeekV4OpenAIThinkingSuffix(info, chat))
			assert.Equal(t, base, chat.Model)
			assert.JSONEq(t, `{"type":"enabled"}`, string(chat.THINKING))
			assert.Equal(t, "high", chat.ReasoningEffort)
			assert.Equal(t, base, info.UpstreamModelName)
			assert.Equal(t, "high", info.ReasoningEffort)

			info = deepSeekLowTestRelayInfo(alias)
			claude := &dto.ClaudeRequest{Model: alias, Thinking: &dto.Thinking{Type: "disabled"}}
			require.NoError(t, applyDeepSeekV4ClaudeThinkingSuffix(info, claude))
			assert.Equal(t, base, claude.Model)
			require.NotNil(t, claude.Thinking)
			assert.Equal(t, "enabled", claude.Thinking.Type)
			assert.JSONEq(t, `{"effort":"high"}`, string(claude.OutputConfig))
			assert.Equal(t, base, info.UpstreamModelName)
			assert.Equal(t, "high", info.ReasoningEffort)

			info = deepSeekLowTestRelayInfo(alias)
			responses := &dto.OpenAIResponsesRequest{Model: alias, Reasoning: &dto.Reasoning{Effort: "none"}}
			applyDeepSeekV4ResponsesThinkingSuffix(info, responses)
			assert.Equal(t, base, responses.Model)
			assert.Equal(t, "high", responses.Reasoning.Effort)
			assert.Equal(t, base, info.UpstreamModelName)
			assert.Equal(t, "high", info.ReasoningEffort)
		})
	}
}

func TestDeepSeekFlashEffortSuffixes(t *testing.T) {
	for _, tc := range []struct {
		model    string
		base     string
		thinking string
		effort   string
	}{
		{"deepseek-flash-none", "deepseek-flash", "disabled", ""},
		{"deepseek-flash-low", "deepseek-flash", "enabled", "low"},
		{"deepseek-flash-max", "deepseek-flash", "enabled", "max"},
		{"deepseek-flash", "deepseek-flash", "enabled", "high"},
		{"deepseek-flash-medium", "deepseek-flash-medium", "enabled", "high"},
	} {
		t.Run(tc.model, func(t *testing.T) {
			chat := &dto.GeneralOpenAIRequest{Model: tc.model, ReasoningEffort: "high", THINKING: []byte(`{"type":"enabled"}`)}
			info := deepSeekLowTestRelayInfo(tc.model)
			require.NoError(t, applyDeepSeekV4OpenAIThinkingSuffix(info, chat))
			assert.Equal(t, tc.base, chat.Model)
			assert.Equal(t, tc.thinking, gjson.GetBytes(chat.THINKING, "type").String())
			assert.Equal(t, tc.effort, chat.ReasoningEffort)

			claude := &dto.ClaudeRequest{Model: tc.model, Thinking: &dto.Thinking{Type: "enabled"}, OutputConfig: []byte(`{"effort":"high"}`)}
			info = deepSeekLowTestRelayInfo(tc.model)
			require.NoError(t, applyDeepSeekV4ClaudeThinkingSuffix(info, claude))
			assert.Equal(t, tc.base, claude.Model)
			require.NotNil(t, claude.Thinking)
			assert.Equal(t, tc.thinking, claude.Thinking.Type)
			assert.Equal(t, tc.effort, gjson.GetBytes(claude.OutputConfig, "effort").String())

			responses := &dto.OpenAIResponsesRequest{Model: tc.model, Reasoning: &dto.Reasoning{Effort: "high"}}
			info = deepSeekLowTestRelayInfo(tc.model)
			applyDeepSeekV4ResponsesThinkingSuffix(info, responses)
			assert.Equal(t, tc.base, responses.Model)
			want := tc.effort
			if tc.thinking == "disabled" {
				want = "none"
			}
			assert.Equal(t, want, responses.Reasoning.Effort)
		})
	}
	chat := &dto.GeneralOpenAIRequest{Model: "deepseek-flash"}
	require.NoError(t, applyDeepSeekV4OpenAIThinkingSuffix(deepSeekLowTestRelayInfo(chat.Model), chat))
	assert.Empty(t, chat.THINKING)
	assert.Empty(t, chat.ReasoningEffort)
}

// The -none/-low/-max tails of deepseek-flash and deepseek-v4-* reach DeepSeek
// as its own thinking switch and effort in each upstream format, with the base
// name; a bare name keeps DeepSeek's default, and a blacklisted name is sent
// verbatim.
func TestThinkingSuffixUpstreamFields(t *testing.T) {
	settings := model_setting.GetGlobalSettings()
	original := append([]string(nil), settings.ThinkingModelBlacklist...)
	t.Cleanup(func() { settings.ThinkingModelBlacklist = original })
	settings.ThinkingModelBlacklist = append(original, "deepseek-v4-pro-max")

	tests := []struct {
		model, upstream, thinking, effort string
	}{
		{model: "deepseek-flash", upstream: "deepseek-flash"},
		{model: "deepseek-flash-none", upstream: "deepseek-flash", thinking: "disabled"},
		{model: "deepseek-flash-low", upstream: "deepseek-flash", thinking: "enabled", effort: "low"},
		{model: "deepseek-flash-max", upstream: "deepseek-flash", thinking: "enabled", effort: "max"},
		{model: "deepseek-v4-pro-none", upstream: "deepseek-v4-pro", thinking: "disabled"},
		{model: "deepseek-v4-pro-low", upstream: "deepseek-v4-pro", thinking: "enabled", effort: "low"},
		{model: "deepseek-v4-pro-max", upstream: "deepseek-v4-pro-max"},
	}
	newInfo := func(model string) *relaycommon.RelayInfo {
		return &relaycommon.RelayInfo{OriginModelName: model, ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: model}}
	}
	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			chat := &dto.GeneralOpenAIRequest{Model: tt.model}
			converted, err := (&Adaptor{}).ConvertOpenAIRequest(nil, newInfo(tt.model), chat)
			require.NoError(t, err)
			chat = converted.(*dto.GeneralOpenAIRequest)
			assert.Equal(t, tt.upstream, chat.Model)
			assert.Equal(t, tt.effort, chat.ReasoningEffort)

			claude := &dto.ClaudeRequest{Model: tt.model}
			require.NoError(t, applyDeepSeekV4ClaudeThinkingSuffix(newInfo(tt.model), claude))
			assert.Equal(t, tt.upstream, claude.Model)

			converted, err = (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, newInfo(tt.model), dto.OpenAIResponsesRequest{Model: tt.model})
			require.NoError(t, err)
			responses := converted.(dto.OpenAIResponsesRequest)
			assert.Equal(t, tt.upstream, responses.Model)

			if tt.thinking == "" {
				assert.Empty(t, chat.THINKING)
				assert.Nil(t, claude.Thinking)
				assert.Nil(t, claude.OutputConfig)
				assert.Nil(t, responses.Reasoning)
				return
			}
			assert.JSONEq(t, `{"type":"`+tt.thinking+`"}`, string(chat.THINKING))
			require.NotNil(t, claude.Thinking)
			assert.Equal(t, tt.thinking, claude.Thinking.Type)
			require.NotNil(t, responses.Reasoning)
			if tt.thinking == "disabled" {
				assert.Nil(t, claude.OutputConfig)
				assert.Equal(t, "none", responses.Reasoning.Effort)
				return
			}
			assert.JSONEq(t, `{"effort":"`+tt.effort+`"}`, string(claude.OutputConfig))
			assert.Equal(t, tt.effort, responses.Reasoning.Effort)
		})
	}
}

// DeepSeek runs Claude Code and Codex searches server-side and bills them as
// Search and Fetch API calls, whatever model name the client sent. On the
// Responses API the completed web_search_call items of the whole response
// replace the items counted while streaming: open_page is a Fetch call, a
// failed call is free, and an empty output keeps the streamed count.
func TestServerSideSearchBillingKeys(t *testing.T) {
	for _, tc := range []struct {
		format types.RelayFormat
		mode   int
	}{
		{types.RelayFormatClaude, constant.RelayModeUnknown},
		{types.RelayFormatOpenAI, constant.RelayModeChatCompletions},
		{types.RelayFormatOpenAIResponses, constant.RelayModeResponses},
	} {
		info := &relaycommon.RelayInfo{RelayFormat: tc.format, RelayMode: tc.mode, ChannelMeta: &relaycommon.ChannelMeta{}}
		(&Adaptor{}).Init(info)
		assert.Equal(t, SearchToolPriceKey, info.WebSearchBillingKey, tc.format)
		assert.Equal(t, FetchToolPriceKey, info.WebFetchBillingKey, tc.format)
		assert.Equal(t, tc.mode == constant.RelayModeResponses, info.VendorToolUsage != nil, tc.format)
	}

	const output = `[
		{"type":"web_search_call","status":"completed","action":{"type":"search","queries":["deepseek","ws_call_id=call_01"]}},
		{"type":"web_search_call","status":"completed","action":{"type":"open_page","url":"https://example.com/"}},
		{"type":"web_search_call","status":"failed","action":{"type":"open_page","url":"https://api-docs.deepseek.com/"}},
		{"type":"message","status":"completed"}]`
	for _, raw := range []string{
		`{"status":"completed","output":` + output + `}`,
		`{"type":"response.completed","response":{"status":"completed","output":` + output + `}}`,
	} {
		info := &relaycommon.RelayInfo{
			OriginModelName: "gpt-5.1-codex",
			RelayFormat:     types.RelayFormatOpenAIResponses,
			RelayMode:       constant.RelayModeResponses,
			ChannelMeta:     &relaycommon.ChannelMeta{},
		}
		(&Adaptor{}).Init(info)
		for range 3 {
			info.CountBillableToolCall(dto.BuildInCallWebSearchCall, "")
		}
		assert.Equal(t, map[string]int{SearchToolPriceKey: 3}, toolCounts(info), "streamed items bill under the Search key")

		info.ApplyVendorToolUsage([]byte(`{"type":"response.failed","response":{"status":"failed","output":[]}}`))
		assert.Equal(t, map[string]int{SearchToolPriceKey: 3}, toolCounts(info), "an empty output keeps the streamed count")

		info.ApplyVendorToolUsage([]byte(raw))
		assert.Equal(t, map[string]int{SearchToolPriceKey: 1, FetchToolPriceKey: 1}, toolCounts(info), raw)
	}
}

func toolCounts(info *relaycommon.RelayInfo) map[string]int {
	counts := map[string]int{}
	for name, tool := range info.ResponsesUsageInfo.BuiltInTools {
		counts[name] = tool.CallCount
	}
	return counts
}

// Explicit @ modifiers reach DeepSeek as its own thinking switch and effort in
// every format and override the client's reasoning fields: no budget_tokens,
// adaptive thinking, max_tokens raise or sampling change. An @ effort wins
// over a -none/-low/-max tail, a budget maps to an effort, and a request
// without a modifier passes through unchanged. The usage log records the
// effort DeepSeek runs at.
func TestModelModifiersUpstreamFields(t *testing.T) {
	type client struct{ chat, claude, responses string }
	tests := []struct {
		name, model      string
		client           client
		thinking, effort string // thinking "" = untouched
		logEffort        string
	}{
		{name: "thinking on", model: "deepseek-flash@thinking:on", thinking: "enabled", logEffort: "high"},
		{name: "thinking off", model: "deepseek-flash@thinking:off", thinking: "disabled", logEffort: "none"},
		{name: "thinking zero budget", model: "deepseek-flash@thinking:0", thinking: "disabled", logEffort: "none"},
		{name: "effort none", model: "deepseek-flash@effort:none", thinking: "disabled", logEffort: "none"},
		{name: "effort low", model: "deepseek-flash@effort:low", thinking: "enabled", effort: "low", logEffort: "low"},
		{name: "effort medium", model: "deepseek-flash@effort:medium", thinking: "enabled", effort: "medium", logEffort: "high"},
		{name: "effort xhigh", model: "deepseek-flash@effort:xhigh", thinking: "enabled", effort: "xhigh", logEffort: "high"},
		{name: "effort max on pro", model: "deepseek-v4-pro@effort:max", thinking: "enabled", effort: "max", logEffort: "max"},
		{name: "budget maps to an effort", model: "deepseek-flash@thinking:8192", thinking: "enabled", effort: "medium", logEffort: "high"},
		{name: "adaptive", model: "deepseek-flash@thinking:adaptive", thinking: "enabled", logEffort: "high"},
		{name: "temperature with effort", model: "deepseek-flash@temperature:0.3@effort:low", thinking: "enabled", effort: "low", logEffort: "low"},
		{name: "effort overrides client thinking off", model: "deepseek-flash@effort:max", thinking: "enabled", effort: "max", logEffort: "max",
			client: client{chat: `"thinking":{"type":"disabled"}`, claude: `"thinking":{"type":"disabled"}`, responses: `"reasoning":{"effort":"none"}`}},
		{name: "thinking off overrides client thinking", model: "deepseek-flash@thinking:off", thinking: "disabled", logEffort: "none",
			client: client{chat: `"thinking":{"type":"enabled"},"reasoning_effort":"high"`,
				claude:    `"thinking":{"type":"enabled","budget_tokens":2048},"output_config":{"effort":"high","format":{"type":"json_schema"}}`,
				responses: `"reasoning":{"effort":"high"}`}},
		{name: "budget overrides client effort none", model: "deepseek-flash@thinking:8192", thinking: "enabled", effort: "medium", logEffort: "high",
			client: client{responses: `"reasoning":{"effort":"none"}`}},
		{name: "@ effort wins over a tail", model: "deepseek-flash-max@effort:low", thinking: "enabled", effort: "low", logEffort: "low"},
		{name: "no modifier", model: "deepseek-flash",
			client: client{chat: `"thinking":{"type":"disabled"},"reasoning_effort":"low"`, claude: `"thinking":{"type":"adaptive"},"output_config":{"effort":"low"}`, responses: `"reasoning":{"effort":"low"}`}},
	}
	extra := func(members string) string {
		if members == "" {
			return ""
		}
		return "," + members
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodies := map[types.RelayFormat]string{
				types.RelayFormatOpenAI:          fmt.Sprintf(`{"model":%q,"messages":[{"role":"user","content":"hi"}]%s}`, tt.model, extra(tt.client.chat)),
				types.RelayFormatClaude:          fmt.Sprintf(`{"model":%q,"max_tokens":64,"temperature":0.7,"messages":[{"role":"user","content":"hi"}]%s}`, tt.model, extra(tt.client.claude)),
				types.RelayFormatOpenAIResponses: fmt.Sprintf(`{"model":%q,"input":"hi"%s}`, tt.model, extra(tt.client.responses)),
			}
			for format, body := range bodies {
				info := &relaycommon.RelayInfo{
					OriginModelName: tt.model,
					RelayFormat:     format,
					ChannelMeta:     &relaycommon.ChannelMeta{UpstreamModelName: tt.model},
				}
				var sent []byte
				switch format {
				case types.RelayFormatOpenAI:
					original, outbound := &dto.GeneralOpenAIRequest{}, &dto.GeneralOpenAIRequest{}
					require.NoError(t, common.UnmarshalJsonStr(body, original))
					require.NoError(t, common.UnmarshalJsonStr(body, outbound))
					info.Request = original
					require.NoError(t, helper.ApplyReasoningModelSuffix(nil, info, outbound))
					converted, err := (&Adaptor{}).ConvertOpenAIRequest(nil, info, outbound)
					require.NoError(t, err)
					sent, err = common.Marshal(converted)
					require.NoError(t, err)
				case types.RelayFormatClaude:
					original, outbound := &dto.ClaudeRequest{}, &dto.ClaudeRequest{}
					require.NoError(t, common.UnmarshalJsonStr(body, original))
					require.NoError(t, common.UnmarshalJsonStr(body, outbound))
					info.Request = original
					require.NoError(t, helper.ApplyReasoningModelSuffix(nil, info, outbound))
					converted, err := (&Adaptor{}).ConvertClaudeRequest(nil, info, outbound)
					require.NoError(t, err)
					sent, err = common.Marshal(converted)
					require.NoError(t, err)
				default:
					original, outbound := &dto.OpenAIResponsesRequest{}, &dto.OpenAIResponsesRequest{}
					require.NoError(t, common.UnmarshalJsonStr(body, original))
					require.NoError(t, common.UnmarshalJsonStr(body, outbound))
					info.Request = original
					require.NoError(t, helper.ApplyReasoningModelSuffix(nil, info, outbound))
					converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(nil, info, *outbound)
					require.NoError(t, err)
					sent, err = common.Marshal(converted)
					require.NoError(t, err)
				}
				wire := gjson.ParseBytes(sent)
				wantModel := "deepseek-flash"
				if strings.HasPrefix(tt.model, "deepseek-v4-pro") {
					wantModel = "deepseek-v4-pro"
				}
				assert.Equal(t, wantModel, wire.Get("model").String(), format)
				if tt.thinking == "" {
					// No modifier: the client's reasoning fields go out as sent.
					for _, member := range []string{"thinking", "reasoning_effort", "output_config", "reasoning"} {
						assert.Equal(t, gjson.Get(body, member).Raw, wire.Get(member).Raw, "%s %s", format, member)
					}
					continue
				}
				switch format {
				case types.RelayFormatOpenAI:
					assert.JSONEq(t, `{"type":"`+tt.thinking+`"}`, wire.Get("thinking").Raw, format)
					assert.Equal(t, tt.effort, wire.Get("reasoning_effort").String(), format)
					assert.False(t, wire.Get("reasoning").Exists(), format)
				case types.RelayFormatClaude:
					assert.JSONEq(t, `{"type":"`+tt.thinking+`"}`, wire.Get("thinking").Raw, format)
					assert.Equal(t, tt.effort, wire.Get("output_config.effort").String(), format)
					assert.Equal(t, gjson.Get(body, "output_config.format").Raw, wire.Get("output_config.format").Raw, format)
					assert.Equal(t, int64(64), wire.Get("max_tokens").Int(), format)
					wantTemperature := 0.7
					if strings.Contains(tt.model, "@temperature:0.3") {
						wantTemperature = 0.3
					}
					assert.Equal(t, wantTemperature, wire.Get("temperature").Float(), format)
				default:
					wantEffort := tt.effort
					if tt.thinking == "disabled" {
						wantEffort = "none"
					}
					assert.Equal(t, wantEffort, wire.Get("reasoning.effort").String(), format)
				}
				assert.Equal(t, tt.logEffort, info.ReasoningEffort, format)
			}
		})
	}
}
