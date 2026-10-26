package moonshot

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	channelconstant "github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/relay/channel/claude"
	"github.com/QuantumNous/new-api/relay/channel/openai"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/setting/reasoning"

	"github.com/gin-gonic/gin"
)

type Adaptor struct {
	kimiFormulaNames []string
}

// PreparePostOverrideRequest consumes the gateway-only kimi_tools marker while
// TextHelper still owns the post-override JSON bytes. The adaptor instance is
// request-scoped, so the parsed Formula selection can be handed to DoRequest
// without reading ordinary Moonshot request bodies.
func (a *Adaptor) PreparePostOverrideRequest(jsonData []byte) ([]byte, error) {
	a.kimiFormulaNames = nil
	var request map[string]json.RawMessage
	if err := common.Unmarshal(jsonData, &request); err != nil {
		return nil, err
	}
	marker, ok := request[kimiToolsField]
	if !ok {
		return jsonData, nil
	}
	delete(request, kimiToolsField)
	formulaNames, err := parseKimiFormulaNames(marker)
	if err != nil {
		return nil, newKimiLoopError("kimi_tool_loop_invalid_tools", http.StatusBadRequest, err.Error())
	}
	cleaned, err := common.Marshal(request)
	if err != nil {
		return nil, err
	}
	if len(formulaNames) > 0 {
		a.kimiFormulaNames = append([]string(nil), formulaNames...)
	}
	return cleaned, nil
}

func (a *Adaptor) ConvertGeminiRequest(*gin.Context, *relaycommon.RelayInfo, *dto.GeminiChatRequest) (any, error) {
	//TODO implement me
	return nil, errors.New("not implemented")
}

func (a *Adaptor) ConvertClaudeRequest(c *gin.Context, info *relaycommon.RelayInfo, req *dto.ClaudeRequest) (any, error) {
	adaptor := claude.Adaptor{}
	return adaptor.ConvertClaudeRequest(c, info, req)
}

func (a *Adaptor) ConvertAudioRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.AudioRequest) (io.Reader, error) {
	//TODO implement me
	return nil, errors.New("not supported")
}

func (a *Adaptor) ConvertImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error) {
	adaptor := openai.Adaptor{}
	return adaptor.ConvertImageRequest(c, info, request)
}

func (a *Adaptor) Init(info *relaycommon.RelayInfo) {
}

func (a *Adaptor) GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	baseURL := info.ChannelBaseUrl
	if specialPlan, ok := channelconstant.ChannelSpecialBases[baseURL]; ok {
		if info.RelayFormat == types.RelayFormatClaude {
			return fmt.Sprintf("%s/v1/messages", specialPlan.ClaudeBaseURL), nil
		}
		if info.RelayFormat == types.RelayFormatOpenAI {
			return fmt.Sprintf("%s/chat/completions", specialPlan.OpenAIBaseURL), nil
		}
	}

	switch info.RelayFormat {
	case types.RelayFormatClaude:
		return fmt.Sprintf("%s/anthropic/v1/messages", info.ChannelBaseUrl), nil
	default:
		if info.RelayMode == constant.RelayModeRerank {
			return fmt.Sprintf("%s/v1/rerank", info.ChannelBaseUrl), nil
		} else if info.RelayMode == constant.RelayModeEmbeddings {
			return fmt.Sprintf("%s/v1/embeddings", info.ChannelBaseUrl), nil
		} else if info.RelayMode == constant.RelayModeChatCompletions {
			return fmt.Sprintf("%s/v1/chat/completions", info.ChannelBaseUrl), nil
		} else if info.RelayMode == constant.RelayModeCompletions {
			return fmt.Sprintf("%s/v1/completions", info.ChannelBaseUrl), nil
		} else if info.RelayMode == constant.RelayModeResponses {
			return fmt.Sprintf("%s/v1/responses", info.ChannelBaseUrl), nil
		}
		return fmt.Sprintf("%s/v1/chat/completions", info.ChannelBaseUrl), nil
	}
}

func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Header, info *relaycommon.RelayInfo) error {
	channel.SetupApiRequestHeader(info, c, req)
	req.Set("Authorization", fmt.Sprintf("Bearer %s", info.ApiKey))
	return nil
}

func (a *Adaptor) ConvertOpenAIRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) (any, error) {
	upstreamModelName := getUpstreamModelName(info, request.Model)
	isKimiK26ThinkingAlias := upstreamModelName == "kimi-k2.6-thinking" ||
		(info != nil && info.OriginModelName == "kimi-k2.6-thinking")
	if isKimiK26ThinkingAlias {
		request.Model = "kimi-k2.6"
		request.THINKING = json.RawMessage(`{"type":"enabled"}`)
		upstreamModelName = request.Model
		if info != nil && info.ChannelMeta != nil {
			info.UpstreamModelName = request.Model
		}
	} else if upstreamModelName == "kimi-k2.6" && len(request.THINKING) == 0 {
		request.THINKING = json.RawMessage(`{"type":"disabled"}`)
	}
	reasoningEffort := ""
	reasoningModel := upstreamModelName
	if base, effort, ok := reasoning.ParseKimiReasoningEffortSuffix(upstreamModelName); ok &&
		(base == "kimi-k3" || base == "kimi-k3.1-pro" || base == "kimi-k3.1-pro-highspeed") {
		reasoningModel, reasoningEffort = base, effort
	} else {
		// A model mapping can replace the effort alias with the model's base
		// name or a pinned snapshot of it, and the suffix then survives only on
		// the origin name. Recover the effort from there, but keep the mapped
		// name so the mapping target is never rewritten back to the base.
		if info != nil && reasoning.IsKimiReasoningEffortModel(upstreamModelName) {
			if originBase, effort, ok := reasoning.ParseKimiReasoningEffortSuffix(info.OriginModelName); ok &&
				strings.HasPrefix(upstreamModelName, originBase) {
				reasoningEffort = effort
			}
		}
	}
	if reasoningEffort != "" {
		request.Model = reasoningModel
		request.ReasoningEffort = reasoningEffort
		upstreamModelName = request.Model
		if info != nil {
			info.ReasoningEffort = request.ReasoningEffort
			if info.ChannelMeta != nil {
				info.UpstreamModelName = request.Model
			}
		}
	}
	if upstreamModelName == "kimi-k3.1-pro-none" || upstreamModelName == "kimi-k3.1-pro-highspeed-none" {
		return nil, newKimiLoopError("invalid_reasoning_effort", http.StatusBadRequest, "Kimi K3.1 Pro supports reasoning_effort low, high, or max")
	}
	if upstreamModelName == "kimi-k3.1-pro" || upstreamModelName == "kimi-k3.1-pro-highspeed" {
		switch request.ReasoningEffort {
		case "", "low", "high", "max":
		default:
			return nil, newKimiLoopError("invalid_reasoning_effort", http.StatusBadRequest, "Kimi K3.1 Pro supports reasoning_effort low, high, or max")
		}
		request.THINKING = nil
		request.FrequencyPenalty = nil
		request.PresencePenalty = nil
	}
	if usesFixedSamplingParameters(upstreamModelName) {
		request.Temperature = nil
		request.TopP = nil
		request.TopK = nil
		request.N = nil
		if request.FrequencyPenalty != nil {
			*request.FrequencyPenalty = 0
		}
		if request.PresencePenalty != nil {
			*request.PresencePenalty = 0
		}
	}
	return request, nil
}

func getUpstreamModelName(info *relaycommon.RelayInfo, fallback string) string {
	if info != nil && info.ChannelMeta != nil && info.UpstreamModelName != "" {
		return info.UpstreamModelName
	}
	return fallback
}

func usesFixedSamplingParameters(model string) bool {
	return strings.EqualFold(model, "kimi-k2.6") || strings.EqualFold(model, "kimi-k3") ||
		model == "kimi-k3.1-pro" || model == "kimi-k3.1-pro-highspeed"
}

// ConvertOpenAIResponsesRequest forwards Responses requests natively.
func (a *Adaptor) ConvertOpenAIResponsesRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.OpenAIResponsesRequest) (any, error) {
	model := getUpstreamModelName(info, request.Model)
	base, _, _ := reasoning.ParseKimiReasoningEffortSuffix(model)
	if base != "kimi-k3.1-pro" && base != "kimi-k3.1-pro-highspeed" &&
		model != "kimi-k3.1-pro" && model != "kimi-k3.1-pro-highspeed" &&
		model != "kimi-k3.1-pro-none" && model != "kimi-k3.1-pro-highspeed-none" {
		return request, nil
	}
	chat := &dto.GeneralOpenAIRequest{Model: request.Model}
	if request.Reasoning != nil {
		chat.ReasoningEffort = request.Reasoning.Effort
	}
	if _, err := a.ConvertOpenAIRequest(c, info, chat); err != nil {
		return nil, err
	}
	request.Model = chat.Model
	if chat.ReasoningEffort != "" {
		if request.Reasoning == nil {
			request.Reasoning = &dto.Reasoning{}
		}
		request.Reasoning.Effort = chat.ReasoningEffort
	}
	request.Temperature, request.TopP = nil, nil
	request.FrequencyPenalty, request.PresencePenalty, request.TopK = nil, nil, nil
	return request, nil
}

func (a *Adaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	if info != nil && info.RelayMode == constant.RelayModeChatCompletions && len(a.kimiFormulaNames) > 0 {
		formulaNames := a.kimiFormulaNames
		a.kimiFormulaNames = nil
		return doKimiFormulaRequest(a, c, info, requestBody, formulaNames)
	}
	return channel.DoApiRequest(a, c, info, requestBody)
}

func (a *Adaptor) ConvertRerankRequest(c *gin.Context, relayMode int, request dto.RerankRequest) (any, error) {
	return request, nil
}

func (a *Adaptor) ConvertEmbeddingRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.EmbeddingRequest) (any, error) {
	return request, nil
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, err *types.NewAPIError) {
	switch info.RelayFormat {
	case types.RelayFormatClaude:
		adaptor := claude.Adaptor{}
		return adaptor.DoResponse(c, resp, info)
	default:
		adaptor := openai.Adaptor{}
		usage, apiErr := adaptor.DoResponse(c, resp, info)
		if parsed, ok := usage.(*dto.Usage); ok && parsed != nil {
			// Managed Formula rounds have already incorporated each response's
			// headers; the final response header describes only its last round.
			if info.KimiToolLoop == nil || len(info.KimiToolLoop.Usages) == 0 {
				applyKimiCacheUsage(parsed, resp.Header)
			}
		}
		return usage, apiErr
	}
}

// applyKimiCacheUsage preserves OpenAI's inclusive input-token semantics while
// carrying Kimi's authoritative TTL counts through the shared cache counters.
func applyKimiCacheUsage(usage *dto.Usage, headers http.Header) {
	usage.UsageSemantic = dto.BillingUsageSemanticOpenAI
	if usage.BillingUsage != nil && usage.BillingUsage.OpenAIUsage != nil {
		defer func() {
			usage.BillingUsage.OpenAIUsage.UsageSemantic = usage.UsageSemantic
			usage.BillingUsage.OpenAIUsage.PromptTokensDetails = usage.PromptTokensDetails.Clone()
			if usage.InputTokensDetails != nil {
				details := usage.InputTokensDetails.Clone()
				usage.BillingUsage.OpenAIUsage.InputTokensDetails = &details
			}
			usage.BillingUsage.OpenAIUsage.ClaudeCacheCreation5mTokens = usage.ClaudeCacheCreation5mTokens
			usage.BillingUsage.OpenAIUsage.ClaudeCacheCreation1hTokens = usage.ClaudeCacheCreation1hTokens
		}()
	}
	short, long := headers.Get("Msh-Usage-Cache-Write-Tokens-5m"), headers.Get("Msh-Usage-Cache-Write-Tokens-1h")
	total := usage.PromptTokensDetails.CacheCreationTokensTotal()
	if short == "" && long == "" {
		if total > 0 && usage.ClaudeCacheCreation5mTokens == 0 && usage.ClaudeCacheCreation1hTokens == 0 {
			common.SysError("Kimi cache-write TTL unavailable; retaining aggregate cache-write billing")
		}
		return
	}
	write5m, err5m := strconv.Atoi(short)
	write1h, err1h := strconv.Atoi(long)
	cached, prompt := usage.PromptTokensDetails.CachedTokens, usage.PromptTokens
	if err5m != nil || err1h != nil || cached < 0 || cached > prompt || write5m < 0 || write1h < 0 ||
		write5m > prompt-cached || write1h > prompt-cached-write5m ||
		write5m+write1h != total {
		common.SysError("invalid Kimi cache-write TTL breakdown; retaining aggregate cache-write billing")
		return
	}
	usage.PromptTokensDetails.CacheWriteTokens = write5m + write1h
	usage.PromptTokensDetails.CacheWriteTokensDetails = nil
	if usage.InputTokensDetails != nil {
		usage.InputTokensDetails.CacheWriteTokens = write5m + write1h
		usage.InputTokensDetails.CacheWriteTokensDetails = nil
	}
	usage.ClaudeCacheCreation5mTokens, usage.ClaudeCacheCreation1hTokens = write5m, write1h
}

func (a *Adaptor) GetModelList() []string {
	return ModelList
}

func (a *Adaptor) GetChannelName() string {
	return ChannelName
}
