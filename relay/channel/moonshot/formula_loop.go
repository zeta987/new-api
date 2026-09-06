package moonshot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	channelconstant "github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/relay/channel"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

const (
	kimiToolsField                = "kimi_tools"
	kimiFormulaNamespace          = "moonshot"
	maxKimiFormulaChatRounds      = 8
	maxKimiFormulaExecutions      = 16
	maxKimiFormulaDeclarations    = 64
	maxKimiManagedRequestBytes    = 8 << 20
	maxKimiDeclarationBodyBytes   = 1 << 20
	maxKimiChatResponseBodyBytes  = 8 << 20
	maxKimiFiberResponseBodyBytes = 8 << 20
	maxKimiToolResultBytes        = 16 << 20
)

var kimiFormulaNames = map[string]struct{}{
	"web-search":  {},
	"fetch":       {},
	"code-runner": {},
}

type kimiFormulaBinding struct {
	formulaName string
	formulaURI  string
}

type kimiFormulaToolsResponse struct {
	Tools []json.RawMessage `json:"tools"`
}

type kimiChatResponse struct {
	Choices []struct {
		Message      json.RawMessage `json:"message"`
		FinishReason string          `json:"finish_reason"`
		Usage        json.RawMessage `json:"usage,omitempty"`
	} `json:"choices"`
	Usage json.RawMessage `json:"usage,omitempty"`
	Error json.RawMessage `json:"error,omitempty"`
}

type kimiAssistantMessage struct {
	ToolCalls []dto.ToolCallResponse `json:"tool_calls,omitempty"`
}

type kimiFiberRequest struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type kimiFiberResponse struct {
	Status  string `json:"status"`
	Context struct {
		Output          json.RawMessage `json:"output"`
		EncryptedOutput *string         `json:"encrypted_output"`
	} `json:"context"`
}

func doKimiFormulaRequest(a *Adaptor, c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader, formulaNames []string) (*http.Response, error) {
	if c == nil || c.Request == nil {
		return nil, kimiLoopError(info, c, "kimi_tool_loop_request_context_missing", http.StatusBadRequest, "managed Kimi request context is missing")
	}
	loopContext, cancelLoop := context.WithTimeout(c.Request.Context(), kimiFormulaLoopTimeout())
	defer cancelLoop()

	requestBytes, err := io.ReadAll(io.LimitReader(requestBody, maxKimiManagedRequestBytes+1))
	if err != nil {
		return nil, kimiLoopError(info, c, "kimi_tool_loop_request_read_failed", http.StatusBadRequest, "failed to read managed Kimi request")
	}
	if len(requestBytes) > maxKimiManagedRequestBytes {
		return nil, types.NewErrorWithStatusCode(errors.New("Moonshot chat request is too large"), types.ErrorCodeInvalidRequest, http.StatusRequestEntityTooLarge, types.ErrOptionWithSkipRetry())
	}
	var request map[string]json.RawMessage
	if err := common.Unmarshal(requestBytes, &request); err != nil {
		return nil, kimiLoopError(info, c, "kimi_tool_loop_invalid_request", http.StatusBadRequest, "managed Kimi request is invalid")
	}
	loopInfo := &relaycommon.KimiToolLoopInfo{
		ToolCalls:          make(map[string]int),
		AttemptedToolCalls: make(map[string]int),
	}
	info.KimiToolLoop = loopInfo

	internalInfo := *info
	internalInfo.IsStream = false
	internalInfo.DisablePing = true
	internalInfo.KimiToolLoop = loopInfo

	clientStream := info.IsStream
	request["stream"] = json.RawMessage("false")
	delete(request, "stream_options")

	var messages []json.RawMessage
	if rawMessages, ok := request["messages"]; ok && len(bytes.TrimSpace(rawMessages)) > 0 && !bytes.Equal(bytes.TrimSpace(rawMessages), []byte("null")) {
		if err := common.Unmarshal(rawMessages, &messages); err != nil {
			return nil, kimiLoopError(info, c, "kimi_tool_loop_invalid_messages", http.StatusBadRequest, "managed Kimi messages are invalid")
		}
	}
	var tools []json.RawMessage
	if rawTools, ok := request["tools"]; ok && len(bytes.TrimSpace(rawTools)) > 0 && !bytes.Equal(bytes.TrimSpace(rawTools), []byte("null")) {
		if err := common.Unmarshal(rawTools, &tools); err != nil {
			return nil, kimiLoopError(info, c, "kimi_tool_loop_invalid_tools", http.StatusBadRequest, "managed Kimi tools are invalid")
		}
	}

	functionBindings, declarations, err := fetchKimiFormulaDeclarations(a, c, loopContext, &internalInfo, info, formulaNames, tools)
	if err != nil {
		return nil, err
	}
	tools = append(tools, declarations...)
	if err := setKimiRequestArray(request, "tools", tools); err != nil {
		return nil, kimiLoopError(info, c, "kimi_tool_loop_invalid_tools", http.StatusBadRequest, "managed Kimi tools are invalid")
	}

	executions := 0
	toolResultBytes := 0
	for round := 1; round <= maxKimiFormulaChatRounds; round++ {
		if err := setKimiRequestArray(request, "messages", messages); err != nil {
			return nil, kimiLoopError(info, c, "kimi_tool_loop_invalid_messages", http.StatusBadRequest, "managed Kimi messages are invalid")
		}
		outboundBody, err := common.Marshal(request)
		if err != nil {
			return nil, kimiLoopError(info, c, "kimi_tool_loop_invalid_request", http.StatusBadRequest, "managed Kimi request is invalid")
		}
		if len(outboundBody) > maxKimiManagedRequestBytes {
			return nil, kimiLoopError(info, c, "kimi_tool_loop_request_too_large", http.StatusRequestEntityTooLarge, "managed Kimi request is too large")
		}
		estimatedPromptTokens := service.CountTextToken(string(outboundBody), info.UpstreamModelName)
		if len(outboundBody) > estimatedPromptTokens {
			estimatedPromptTokens = len(outboundBody)
		}
		maxCompletionTokens := kimiMaxCompletionTokens(request)
		nextEstimate, estimateErr := service.EstimateKimiToolLoopQuotaChecked(info, estimatedPromptTokens, maxCompletionTokens)
		if estimateErr != nil {
			return nil, kimiLoopExistingError(info, c, "kimi_tool_loop_quota_estimate_failed", estimateErr)
		}
		if reserveErr := service.ReserveKimiToolLoopQuota(c, info, nextEstimate); reserveErr != nil {
			return nil, kimiLoopExistingError(info, c, "kimi_tool_loop_quota_reserve_failed", reserveErr)
		}

		chatURL, err := a.GetRequestURL(&internalInfo)
		if err != nil {
			return nil, kimiLoopError(info, c, "kimi_tool_loop_chat_url_failed", http.StatusInternalServerError, "managed Kimi chat URL is invalid")
		}
		resp, err := channel.DoApiRequestWithContext(a, c, &internalInfo, loopContext, http.MethodPost, chatURL, bytes.NewReader(outboundBody))
		if err != nil {
			return nil, kimiLoopError(info, c, "kimi_tool_loop_chat_failed", http.StatusBadGateway, "managed Kimi chat request failed")
		}
		responseBody, readErr := readKimiResponseBody(resp, maxKimiChatResponseBodyBytes)
		if readErr != nil {
			return nil, kimiLoopError(info, c, "kimi_tool_loop_chat_response_failed", http.StatusBadGateway, "managed Kimi chat response is invalid")
		}
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			return nil, kimiLoopError(info, c, "kimi_tool_loop_chat_status", http.StatusBadGateway, "managed Kimi chat request failed")
		}

		chatResponse, usage, err := parseKimiChatResponse(responseBody)
		if usage != nil {
			loopInfo.Usages = append(loopInfo.Usages, *usage)
		}
		if err != nil {
			return nil, kimiLoopError(info, c, "kimi_tool_loop_chat_response_failed", http.StatusBadGateway, "managed Kimi chat response is invalid")
		}
		aggregatedUsage := aggregateKimiUsages(loopInfo.Usages)

		choice := chatResponse.Choices[0]
		var assistant kimiAssistantMessage
		if err := common.Unmarshal(choice.Message, &assistant); err != nil {
			return nil, kimiLoopError(info, c, "kimi_tool_loop_chat_response_failed", http.StatusBadGateway, "managed Kimi assistant message is invalid")
		}
		if len(assistant.ToolCalls) == 0 {
			if choice.FinishReason == "tool_calls" {
				return nil, kimiLoopError(info, c, "kimi_tool_loop_invalid_tool_batch", http.StatusBadGateway, "managed Kimi tool-call response is invalid")
			}
			loopInfo.Completed = true
			finalBody, err := buildKimiFinalResponse(responseBody, aggregatedUsage, clientStream)
			if err != nil {
				return nil, kimiLoopError(info, c, "kimi_tool_loop_final_response_failed", http.StatusInternalServerError, "managed Kimi final response is invalid")
			}
			return replaceKimiResponseBody(resp, finalBody, clientStream), nil
		}
		if err := validateKimiToolBatch(choice.FinishReason, assistant.ToolCalls); err != nil {
			return nil, kimiLoopError(info, c, "kimi_tool_loop_invalid_tool_batch", http.StatusBadGateway, "managed Kimi tool-call response is invalid")
		}

		allManaged := true
		for _, toolCall := range assistant.ToolCalls {
			if _, ok := functionBindings[toolCall.Function.Name]; !ok {
				allManaged = false
				break
			}
		}
		if !allManaged {
			loopInfo.Completed = true
			finalBody, err := buildKimiFinalResponse(responseBody, aggregatedUsage, clientStream)
			if err != nil {
				return nil, kimiLoopError(info, c, "kimi_tool_loop_final_response_failed", http.StatusInternalServerError, "managed Kimi final response is invalid")
			}
			return replaceKimiResponseBody(resp, finalBody, clientStream), nil
		}
		if executions+len(assistant.ToolCalls) > maxKimiFormulaExecutions {
			return nil, kimiLoopError(info, c, "kimi_tool_loop_execution_limit", http.StatusBadGateway, "managed Kimi tool execution limit reached")
		}
		if round == maxKimiFormulaChatRounds {
			return nil, kimiLoopError(info, c, "kimi_tool_loop_round_limit", http.StatusBadGateway, "managed Kimi chat round limit reached")
		}

		messages = append(messages, append(json.RawMessage(nil), choice.Message...))
		for _, toolCall := range assistant.ToolCalls {
			binding := functionBindings[toolCall.Function.Name]
			loopInfo.AttemptedToolCalls[binding.formulaName]++
			if reserveErr := service.ReserveKimiToolLoopQuota(c, info, 0); reserveErr != nil {
				loopInfo.AttemptedToolCalls[binding.formulaName]--
				return nil, kimiLoopExistingError(info, c, "kimi_tool_loop_quota_reserve_failed", reserveErr)
			}
			executions++
			result, succeeded, executeErr := executeKimiFormula(a, c, loopContext, &internalInfo, binding, toolCall)
			if succeeded {
				loopInfo.ToolCalls[binding.formulaName]++
			}
			if executeErr != nil {
				return nil, kimiLoopExistingError(info, c, string(executeErr.GetErrorCode()), executeErr)
			}
			toolResultBytes += len(result)
			if toolResultBytes > maxKimiToolResultBytes {
				return nil, kimiLoopError(info, c, "kimi_tool_loop_result_limit", http.StatusBadGateway, "managed Kimi tool result limit reached")
			}
			message, err := common.Marshal(map[string]any{
				"role":         "tool",
				"tool_call_id": toolCall.ID,
				"content":      result,
			})
			if err != nil {
				return nil, kimiLoopError(info, c, "kimi_tool_loop_result_failed", http.StatusInternalServerError, "managed Kimi tool result is invalid")
			}
			messages = append(messages, message)
		}
	}

	return nil, kimiLoopError(info, c, "kimi_tool_loop_round_limit", http.StatusBadGateway, "managed Kimi chat round limit reached")
}

func kimiFormulaLoopTimeout() time.Duration {
	const fallback = 300 * time.Second
	seconds := channelconstant.StreamingTimeout
	maxSeconds := int64(^uint64(0)>>1) / int64(time.Second)
	if seconds <= 0 || int64(seconds) > maxSeconds {
		return fallback
	}
	return time.Duration(seconds) * time.Second
}

func parseKimiFormulaNames(marker json.RawMessage) ([]string, error) {
	trimmed := bytes.TrimSpace(marker)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil, nil
	}
	var values []string
	if err := common.Unmarshal(trimmed, &values); err != nil {
		return nil, errors.New("kimi_tools must be an array")
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if _, ok := kimiFormulaNames[value]; !ok {
			return nil, errors.New("kimi_tools contains an unsupported managed tool")
		}
		if _, duplicate := seen[value]; duplicate {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result, nil
}

func fetchKimiFormulaDeclarations(a *Adaptor, c *gin.Context, requestContext context.Context, transportInfo, rootInfo *relaycommon.RelayInfo, formulaNames []string, clientTools []json.RawMessage) (map[string]kimiFormulaBinding, []json.RawMessage, error) {
	bindings := make(map[string]kimiFormulaBinding)
	for _, rawTool := range clientTools {
		name, isFunction, err := kimiFunctionName(rawTool)
		if err != nil {
			return nil, nil, kimiLoopError(rootInfo, c, "kimi_tool_loop_invalid_tools", http.StatusBadRequest, "managed Kimi tools are invalid")
		}
		if !isFunction || name == "" {
			continue
		}
		if _, duplicate := bindings[name]; duplicate {
			return nil, nil, kimiLoopError(rootInfo, c, "kimi_tool_loop_tool_collision", http.StatusBadRequest, "managed Kimi tool names must be unique")
		}
		bindings[name] = kimiFormulaBinding{}
	}

	declarations := make([]json.RawMessage, 0, len(formulaNames))
	for _, formulaName := range formulaNames {
		formulaURI := kimiFormulaNamespace + "/" + formulaName + ":latest"
		endpoint, err := kimiFormulaEndpoint(a, transportInfo, formulaURI, "tools")
		if err != nil {
			return nil, nil, kimiLoopError(rootInfo, c, "kimi_tool_loop_formula_url_failed", http.StatusInternalServerError, "managed Kimi Formula URL is invalid")
		}
		resp, err := channel.DoApiRequestWithContext(a, c, transportInfo, requestContext, http.MethodGet, endpoint, bytes.NewReader(nil))
		if err != nil {
			return nil, nil, kimiLoopError(rootInfo, c, "kimi_tool_loop_declaration_failed", http.StatusBadGateway, "managed Kimi tool declaration request failed")
		}
		body, readErr := readKimiResponseBody(resp, maxKimiDeclarationBodyBytes)
		if readErr != nil || resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			return nil, nil, kimiLoopError(rootInfo, c, "kimi_tool_loop_declaration_failed", http.StatusBadGateway, "managed Kimi tool declaration request failed")
		}
		var payload kimiFormulaToolsResponse
		if err := common.Unmarshal(body, &payload); err != nil || len(payload.Tools) == 0 {
			return nil, nil, kimiLoopError(rootInfo, c, "kimi_tool_loop_declaration_invalid", http.StatusBadGateway, "managed Kimi tool declaration response is invalid")
		}
		if len(declarations)+len(payload.Tools) > maxKimiFormulaDeclarations {
			return nil, nil, kimiLoopError(rootInfo, c, "kimi_tool_loop_declaration_limit", http.StatusBadGateway, "managed Kimi tool declaration limit reached")
		}
		for _, declaration := range payload.Tools {
			name, isFunction, err := kimiFunctionName(declaration)
			if err != nil || !isFunction || name == "" {
				return nil, nil, kimiLoopError(rootInfo, c, "kimi_tool_loop_declaration_invalid", http.StatusBadGateway, "managed Kimi tool declaration response is invalid")
			}
			if _, duplicate := bindings[name]; duplicate {
				return nil, nil, kimiLoopError(rootInfo, c, "kimi_tool_loop_tool_collision", http.StatusBadRequest, "managed Kimi tool names must be unique")
			}
			bindings[name] = kimiFormulaBinding{formulaName: formulaName, formulaURI: formulaURI}
			declarations = append(declarations, append(json.RawMessage(nil), declaration...))
		}
	}
	for name, binding := range bindings {
		if binding.formulaURI == "" {
			delete(bindings, name)
		}
	}
	return bindings, declarations, nil
}

func kimiFunctionName(rawTool json.RawMessage) (string, bool, error) {
	var tool struct {
		Type     string `json:"type"`
		Function struct {
			Name string `json:"name"`
		} `json:"function"`
	}
	if err := common.Unmarshal(rawTool, &tool); err != nil {
		return "", false, err
	}
	if tool.Type != "function" {
		return "", false, nil
	}
	return strings.TrimSpace(tool.Function.Name), true, nil
}

func validateKimiToolBatch(finishReason string, toolCalls []dto.ToolCallResponse) error {
	if finishReason != "tool_calls" {
		return errors.New("tool calls require tool_calls finish reason")
	}
	seenIDs := make(map[string]struct{}, len(toolCalls))
	for _, toolCall := range toolCalls {
		callType, ok := toolCall.Type.(string)
		if !ok || callType != "function" || strings.TrimSpace(toolCall.Function.Name) == "" || strings.TrimSpace(toolCall.ID) == "" {
			return errors.New("tool call is incomplete")
		}
		if _, duplicate := seenIDs[toolCall.ID]; duplicate {
			return errors.New("tool call IDs must be unique")
		}
		seenIDs[toolCall.ID] = struct{}{}
	}
	return nil
}

func kimiFormulaEndpoint(a *Adaptor, info *relaycommon.RelayInfo, formulaURI, operation string) (string, error) {
	chatURL, err := a.GetRequestURL(info)
	if err != nil {
		return "", err
	}
	parsed, err := url.Parse(chatURL)
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("upstream URL has no origin")
	}
	parsed.Path = "/v1/formulas/" + formulaURI + "/" + operation
	parsed.RawPath = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func executeKimiFormula(a *Adaptor, c *gin.Context, requestContext context.Context, transportInfo *relaycommon.RelayInfo, binding kimiFormulaBinding, toolCall dto.ToolCallResponse) (string, bool, *types.NewAPIError) {
	endpoint, err := kimiFormulaEndpoint(a, transportInfo, binding.formulaURI, "fibers")
	if err != nil {
		return "", false, newKimiLoopError("kimi_tool_loop_formula_url_failed", http.StatusInternalServerError, "managed Kimi Formula URL is invalid")
	}
	payload, err := common.Marshal(kimiFiberRequest{Name: toolCall.Function.Name, Arguments: toolCall.Function.Arguments})
	if err != nil {
		return "", false, newKimiLoopError("kimi_tool_loop_fiber_request_failed", http.StatusInternalServerError, "managed Kimi tool request is invalid")
	}
	resp, err := channel.DoApiRequestWithContext(a, c, transportInfo, requestContext, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", false, newKimiLoopError("kimi_tool_loop_fiber_failed", http.StatusBadGateway, "managed Kimi tool request failed")
	}
	body, readErr := readKimiResponseBody(resp, maxKimiFiberResponseBodyBytes)
	if readErr != nil || resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", false, newKimiLoopError("kimi_tool_loop_fiber_failed", http.StatusBadGateway, "managed Kimi tool request failed")
	}
	var fiber kimiFiberResponse
	if err := common.Unmarshal(body, &fiber); err != nil {
		return "", false, newKimiLoopError("kimi_tool_loop_fiber_invalid", http.StatusBadGateway, "managed Kimi tool response is invalid")
	}
	if fiber.Status != "succeeded" {
		return "", false, newKimiLoopError("kimi_tool_loop_fiber_unsuccessful", http.StatusBadGateway, "managed Kimi tool execution was unsuccessful")
	}
	trimmedOutput := bytes.TrimSpace(fiber.Context.Output)
	if len(trimmedOutput) > 0 && !bytes.Equal(trimmedOutput, []byte("null")) {
		return common.JsonRawMessageToString(trimmedOutput), true, nil
	}
	if fiber.Context.EncryptedOutput != nil {
		return *fiber.Context.EncryptedOutput, true, nil
	}
	return "", true, newKimiLoopError("kimi_tool_loop_fiber_missing_output", http.StatusBadGateway, "managed Kimi tool response has no output")
}

func readKimiResponseBody(resp *http.Response, limit int64) ([]byte, error) {
	if resp == nil || resp.Body == nil {
		return nil, errors.New("upstream response is missing")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, errors.New("upstream response is too large")
	}
	return body, nil
}

func parseKimiChatResponse(body []byte) (*kimiChatResponse, *dto.Usage, error) {
	var response kimiChatResponse
	if err := common.Unmarshal(body, &response); err != nil {
		return nil, nil, err
	}
	usageRaw := response.Usage
	if !kimiUsagePresent(usageRaw) && len(response.Choices) > 0 {
		usageRaw = response.Choices[0].Usage
	}
	usage, usageErr := normalizeKimiUsage(usageRaw)
	var usagePtr *dto.Usage
	if usageErr == nil {
		usagePtr = &usage
	}
	if len(bytes.TrimSpace(response.Error)) > 0 && !bytes.Equal(bytes.TrimSpace(response.Error), []byte("null")) {
		return nil, usagePtr, errors.New("upstream returned an error")
	}
	if len(response.Choices) != 1 || len(bytes.TrimSpace(response.Choices[0].Message)) == 0 {
		return nil, usagePtr, errors.New("upstream returned no single assistant message")
	}
	if usageErr != nil {
		return nil, nil, usageErr
	}
	return &response, usagePtr, nil
}

func kimiUsagePresent(raw json.RawMessage) bool {
	trimmed := bytes.TrimSpace(raw)
	return len(trimmed) > 0 && !bytes.Equal(trimmed, []byte("null")) && !bytes.Equal(trimmed, []byte("{}"))
}

func normalizeKimiUsage(raw json.RawMessage) (dto.Usage, error) {
	if !kimiUsagePresent(raw) {
		return dto.Usage{}, errors.New("upstream returned no usage")
	}
	var usage dto.Usage
	if err := common.Unmarshal(raw, &usage); err != nil {
		return dto.Usage{}, err
	}
	var alternatives struct {
		CachedTokens         *int `json:"cached_tokens"`
		PromptCacheHitTokens *int `json:"prompt_cache_hit_tokens"`
		InputTokensDetails   *struct {
			CachedTokens int `json:"cached_tokens"`
		} `json:"input_tokens_details"`
	}
	if err := common.Unmarshal(raw, &alternatives); err != nil {
		return dto.Usage{}, err
	}
	if (alternatives.CachedTokens != nil && *alternatives.CachedTokens < 0) ||
		(alternatives.PromptCacheHitTokens != nil && *alternatives.PromptCacheHitTokens < 0) ||
		(alternatives.InputTokensDetails != nil && alternatives.InputTokensDetails.CachedTokens < 0) {
		return dto.Usage{}, errors.New("upstream usage contains a negative token count")
	}
	if usage.PromptTokensDetails.CachedTokens == 0 {
		switch {
		case alternatives.InputTokensDetails != nil && alternatives.InputTokensDetails.CachedTokens > 0:
			usage.PromptTokensDetails.CachedTokens = alternatives.InputTokensDetails.CachedTokens
		case alternatives.CachedTokens != nil && *alternatives.CachedTokens > 0:
			usage.PromptTokensDetails.CachedTokens = *alternatives.CachedTokens
		case alternatives.PromptCacheHitTokens != nil && *alternatives.PromptCacheHitTokens > 0:
			usage.PromptTokensDetails.CachedTokens = *alternatives.PromptCacheHitTokens
		}
	}
	if usage.TotalTokens == 0 {
		usage.TotalTokens = saturatingKimiAdd(usage.PromptTokens, usage.CompletionTokens)
	}
	if !validKimiUsage(usage) {
		return dto.Usage{}, errors.New("upstream usage contains a negative token count")
	}
	return usage, nil
}

func validKimiUsage(usage dto.Usage) bool {
	values := []int{
		usage.PromptTokens,
		usage.CompletionTokens,
		usage.TotalTokens,
		usage.PromptCacheHitTokens,
		usage.InputTokens,
		usage.OutputTokens,
		usage.PromptTokensDetails.CachedTokens,
		usage.PromptTokensDetails.CachedCreationTokens,
		usage.PromptTokensDetails.CacheWriteTokens,
		usage.PromptTokensDetails.TextTokens,
		usage.PromptTokensDetails.AudioTokens,
		usage.PromptTokensDetails.ImageTokens,
		usage.CompletionTokenDetails.TextTokens,
		usage.CompletionTokenDetails.AudioTokens,
		usage.CompletionTokenDetails.ImageTokens,
		usage.CompletionTokenDetails.ReasoningTokens,
		usage.ClaudeCacheCreation5mTokens,
		usage.ClaudeCacheCreation1hTokens,
	}
	if usage.InputTokensDetails != nil {
		values = append(values,
			usage.InputTokensDetails.CachedTokens,
			usage.InputTokensDetails.CachedCreationTokens,
			usage.InputTokensDetails.CacheWriteTokens,
			usage.InputTokensDetails.TextTokens,
			usage.InputTokensDetails.AudioTokens,
			usage.InputTokensDetails.ImageTokens,
		)
	}
	for _, value := range values {
		if value < 0 {
			return false
		}
	}
	return true
}

func aggregateKimiUsages(usages []dto.Usage) dto.Usage {
	var total dto.Usage
	for _, usage := range usages {
		total.PromptTokens = saturatingKimiAdd(total.PromptTokens, usage.PromptTokens)
		total.CompletionTokens = saturatingKimiAdd(total.CompletionTokens, usage.CompletionTokens)
		total.TotalTokens = saturatingKimiAdd(total.TotalTokens, usage.TotalTokens)
		total.PromptCacheHitTokens = saturatingKimiAdd(total.PromptCacheHitTokens, usage.PromptCacheHitTokens)
		total.InputTokens = saturatingKimiAdd(total.InputTokens, usage.InputTokens)
		total.OutputTokens = saturatingKimiAdd(total.OutputTokens, usage.OutputTokens)
		total.PromptTokensDetails.CachedTokens = saturatingKimiAdd(total.PromptTokensDetails.CachedTokens, usage.PromptTokensDetails.CachedTokens)
		total.PromptTokensDetails.CachedCreationTokens = saturatingKimiAdd(total.PromptTokensDetails.CachedCreationTokens, usage.PromptTokensDetails.CachedCreationTokens)
		total.PromptTokensDetails.CacheWriteTokens = saturatingKimiAdd(total.PromptTokensDetails.CacheWriteTokens, usage.PromptTokensDetails.CacheWriteTokens)
		total.PromptTokensDetails.TextTokens = saturatingKimiAdd(total.PromptTokensDetails.TextTokens, usage.PromptTokensDetails.TextTokens)
		total.PromptTokensDetails.AudioTokens = saturatingKimiAdd(total.PromptTokensDetails.AudioTokens, usage.PromptTokensDetails.AudioTokens)
		total.PromptTokensDetails.ImageTokens = saturatingKimiAdd(total.PromptTokensDetails.ImageTokens, usage.PromptTokensDetails.ImageTokens)
		total.CompletionTokenDetails.TextTokens = saturatingKimiAdd(total.CompletionTokenDetails.TextTokens, usage.CompletionTokenDetails.TextTokens)
		total.CompletionTokenDetails.AudioTokens = saturatingKimiAdd(total.CompletionTokenDetails.AudioTokens, usage.CompletionTokenDetails.AudioTokens)
		total.CompletionTokenDetails.ImageTokens = saturatingKimiAdd(total.CompletionTokenDetails.ImageTokens, usage.CompletionTokenDetails.ImageTokens)
		total.CompletionTokenDetails.ReasoningTokens = saturatingKimiAdd(total.CompletionTokenDetails.ReasoningTokens, usage.CompletionTokenDetails.ReasoningTokens)
		total.ClaudeCacheCreation5mTokens = saturatingKimiAdd(total.ClaudeCacheCreation5mTokens, usage.ClaudeCacheCreation5mTokens)
		total.ClaudeCacheCreation1hTokens = saturatingKimiAdd(total.ClaudeCacheCreation1hTokens, usage.ClaudeCacheCreation1hTokens)
		if usage.InputTokensDetails != nil {
			if total.InputTokensDetails == nil {
				total.InputTokensDetails = &dto.InputTokenDetails{}
			}
			total.InputTokensDetails.CachedTokens = saturatingKimiAdd(total.InputTokensDetails.CachedTokens, usage.InputTokensDetails.CachedTokens)
			total.InputTokensDetails.CachedCreationTokens = saturatingKimiAdd(total.InputTokensDetails.CachedCreationTokens, usage.InputTokensDetails.CachedCreationTokens)
			total.InputTokensDetails.CacheWriteTokens = saturatingKimiAdd(total.InputTokensDetails.CacheWriteTokens, usage.InputTokensDetails.CacheWriteTokens)
			total.InputTokensDetails.TextTokens = saturatingKimiAdd(total.InputTokensDetails.TextTokens, usage.InputTokensDetails.TextTokens)
			total.InputTokensDetails.AudioTokens = saturatingKimiAdd(total.InputTokensDetails.AudioTokens, usage.InputTokensDetails.AudioTokens)
			total.InputTokensDetails.ImageTokens = saturatingKimiAdd(total.InputTokensDetails.ImageTokens, usage.InputTokensDetails.ImageTokens)
		}
	}
	return total
}

func saturatingKimiAdd(left, right int) int {
	if right <= 0 {
		return left
	}
	maxInt := int(^uint(0) >> 1)
	if left > maxInt-right {
		return maxInt
	}
	return left + right
}

func buildKimiFinalResponse(body []byte, usage dto.Usage, stream bool) ([]byte, error) {
	var response map[string]json.RawMessage
	if err := common.Unmarshal(body, &response); err != nil {
		return nil, err
	}
	usageBytes, err := common.Marshal(usage)
	if err != nil {
		return nil, err
	}
	var usageMap map[string]json.RawMessage
	if err := common.Unmarshal(usageBytes, &usageMap); err != nil {
		return nil, err
	}
	cachedBytes, err := common.Marshal(usage.PromptTokensDetails.CachedTokens)
	if err != nil {
		return nil, err
	}
	usageMap["cached_tokens"] = cachedBytes
	usageBytes, err = common.Marshal(usageMap)
	if err != nil {
		return nil, err
	}
	response["usage"] = usageBytes
	if !stream {
		return common.Marshal(response)
	}

	var choices []map[string]json.RawMessage
	if err := common.Unmarshal(response["choices"], &choices); err != nil {
		return nil, err
	}
	reasoningChoices := make([]map[string]json.RawMessage, 0, len(choices))
	for _, choice := range choices {
		message, ok := choice["message"]
		if !ok {
			return nil, errors.New("final response has no assistant message")
		}
		var messageMap map[string]json.RawMessage
		if err := common.Unmarshal(message, &messageMap); err != nil {
			return nil, err
		}
		if rawToolCalls, ok := messageMap["tool_calls"]; ok {
			var toolCalls []map[string]json.RawMessage
			if err := common.Unmarshal(rawToolCalls, &toolCalls); err != nil {
				return nil, err
			}
			for index, toolCall := range toolCalls {
				if _, hasIndex := toolCall["index"]; hasIndex {
					continue
				}
				indexBytes, err := common.Marshal(index)
				if err != nil {
					return nil, err
				}
				toolCall["index"] = indexBytes
			}
			indexedCalls, err := common.Marshal(toolCalls)
			if err != nil {
				return nil, err
			}
			messageMap["tool_calls"] = indexedCalls
		}
		reasoningMessage := make(map[string]json.RawMessage, 3)
		if role, ok := messageMap["role"]; ok {
			reasoningMessage["role"] = role
		}
		for _, key := range []string{"reasoning_content", "reasoning"} {
			value, ok := messageMap[key]
			if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
				continue
			}
			reasoningMessage[key] = value
			delete(messageMap, key)
		}
		if len(reasoningMessage) > 1 {
			delete(messageMap, "role")
			reasoningBytes, err := common.Marshal(reasoningMessage)
			if err != nil {
				return nil, err
			}
			reasoningChoice := make(map[string]json.RawMessage, len(choice))
			for key, value := range choice {
				reasoningChoice[key] = value
			}
			reasoningChoice["delta"] = reasoningBytes
			reasoningChoice["finish_reason"] = json.RawMessage("null")
			delete(reasoningChoice, "message")
			reasoningChoices = append(reasoningChoices, reasoningChoice)
		}
		message, err = common.Marshal(messageMap)
		if err != nil {
			return nil, err
		}
		choice["delta"] = message
		delete(choice, "message")
	}
	choiceBytes, err := common.Marshal(choices)
	if err != nil {
		return nil, err
	}
	response["choices"] = choiceBytes
	response["object"] = json.RawMessage(`"chat.completion.chunk"`)
	payloadChunks := make([][]byte, 0, 2)
	if len(reasoningChoices) > 0 {
		reasoningChoiceBytes, err := common.Marshal(reasoningChoices)
		if err != nil {
			return nil, err
		}
		reasoningFrame := make(map[string]json.RawMessage, len(response)-1)
		for key, value := range response {
			if key != "usage" {
				reasoningFrame[key] = value
			}
		}
		reasoningFrame["choices"] = reasoningChoiceBytes
		reasoningChunk, err := common.Marshal(reasoningFrame)
		if err != nil {
			return nil, err
		}
		payloadChunks = append(payloadChunks, reasoningChunk)
	}
	payloadFrame := make(map[string]json.RawMessage, len(response)-1)
	for key, value := range response {
		if key != "usage" {
			payloadFrame[key] = value
		}
	}
	payloadChunk, err := common.Marshal(payloadFrame)
	if err != nil {
		return nil, err
	}
	payloadChunks = append(payloadChunks, payloadChunk)
	response["choices"] = json.RawMessage("[]")
	usageChunk, err := common.Marshal(response)
	if err != nil {
		return nil, err
	}
	var streamBody strings.Builder
	for _, payload := range payloadChunks {
		streamBody.WriteString("data: ")
		streamBody.Write(payload)
		streamBody.WriteString("\n\n")
	}
	streamBody.WriteString("data: ")
	streamBody.Write(usageChunk)
	streamBody.WriteString("\n\ndata: [DONE]\n\n")
	return []byte(streamBody.String()), nil
}

func replaceKimiResponseBody(resp *http.Response, body []byte, stream bool) *http.Response {
	resp.Body = io.NopCloser(bytes.NewReader(body))
	resp.ContentLength = int64(len(body))
	resp.Header.Del("Content-Length")
	if stream {
		resp.Header.Set("Content-Type", "text/event-stream")
	} else {
		resp.Header.Set("Content-Type", "application/json")
	}
	return resp
}

func setKimiRequestArray(request map[string]json.RawMessage, key string, values []json.RawMessage) error {
	data, err := common.Marshal(values)
	if err != nil {
		return err
	}
	request[key] = data
	return nil
}

func kimiMaxCompletionTokens(request map[string]json.RawMessage) int {
	var maxCompletion uint64
	for _, key := range []string{"max_completion_tokens", "max_tokens"} {
		raw, ok := request[key]
		if !ok {
			continue
		}
		var value uint64
		if err := common.Unmarshal(raw, &value); err == nil && value > 0 {
			maxCompletion = value
			break
		}
	}
	maxInt := uint64(^uint(0) >> 1)
	if maxCompletion > maxInt {
		return int(maxInt)
	}
	return int(maxCompletion)
}

func kimiLoopError(info *relaycommon.RelayInfo, c *gin.Context, code string, status int, message string) *types.NewAPIError {
	apiErr := newKimiLoopError(code, status, message)
	return kimiLoopExistingError(info, c, code, apiErr)
}

func newKimiLoopError(code string, status int, message string) *types.NewAPIError {
	return types.NewErrorWithStatusCode(errors.New(message), types.ErrorCode(code), status, types.ErrOptionWithSkipRetry())
}

func kimiLoopExistingError(info *relaycommon.RelayInfo, c *gin.Context, code string, apiErr *types.NewAPIError) *types.NewAPIError {
	if info != nil && info.KimiToolLoop != nil {
		info.KimiToolLoop.Completed = false
		info.KimiToolLoop.ErrorCode = code
		if len(info.KimiToolLoop.Usages) > 0 {
			usage := aggregateKimiUsages(info.KimiToolLoop.Usages)
			service.PostTextConsumeQuota(c, info, &usage, nil)
		}
	}
	return apiErr
}
