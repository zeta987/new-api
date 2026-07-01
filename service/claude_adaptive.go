package service

import (
	"encoding/json"
	"fmt"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/setting/reasoning"
)

const defaultClaudeAdaptiveEffort = "high"

func SetClaudeAdaptiveEffort(request *dto.ClaudeRequest, effort string) bool {
	if request == nil || !reasoning.IsClaudeEffortLevel(effort) {
		return false
	}
	request.Thinking = &dto.Thinking{
		Type:    "adaptive",
		Display: "summarized",
	}
	request.OutputConfig = json.RawMessage(fmt.Sprintf(`{"effort":"%s"}`, effort))
	clearClaudeSamplingParams(request)
	return true
}

func NormalizeClaudePost46AdaptiveRequest(request *dto.ClaudeRequest) {
	if request == nil || !reasoning.IsClaudePost46AdaptiveThinkingModel(request.Model) {
		return
	}
	clearClaudeSamplingParams(request)
	if request.Thinking == nil {
		return
	}

	switch request.Thinking.Type {
	case "enabled":
		request.Thinking.Type = "adaptive"
		request.Thinking.BudgetTokens = nil
		if request.Thinking.Display == "" {
			request.Thinking.Display = "summarized"
		}
		if !reasoning.IsClaudeEffortLevel(request.GetEfforts()) {
			request.OutputConfig = json.RawMessage(fmt.Sprintf(`{"effort":"%s"}`, defaultClaudeAdaptiveEffort))
		}
	case "adaptive":
		request.Thinking.BudgetTokens = nil
		if request.Thinking.Display == "" {
			request.Thinking.Display = "summarized"
		}
		if effort := request.GetEfforts(); effort != "" && !reasoning.IsClaudeEffortLevel(effort) {
			request.OutputConfig = json.RawMessage(fmt.Sprintf(`{"effort":"%s"}`, defaultClaudeAdaptiveEffort))
		}
	}
}

func clearClaudeSamplingParams(request *dto.ClaudeRequest) {
	request.Temperature = nil
	request.TopP = nil
	request.TopK = nil
}
