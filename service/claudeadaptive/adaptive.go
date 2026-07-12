package claudeadaptive

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/setting/reasoning"
)

const defaultEffort = "high"

// ApplyEffortSuffix converts a supported Claude model effort suffix into the
// upstream request fields expected by that model generation.
func ApplyEffortSuffix(request *dto.ClaudeRequest) bool {
	if request == nil {
		return false
	}
	baseModel, effort, ok := reasoning.TrimEffortSuffix(request.Model)
	if !ok || effort == "" ||
		!reasoning.IsClaudeAdaptiveThinkingModel(baseModel) ||
		!reasoning.IsClaudeEffortLevel(effort) {
		return false
	}

	request.Model = baseModel
	if reasoning.IsClaudePost46AdaptiveThinkingModel(baseModel) {
		return SetEffort(request, effort)
	}

	request.Thinking = &dto.Thinking{Type: "adaptive"}
	request.OutputConfig = json.RawMessage(fmt.Sprintf(`{"effort":"%s"}`, effort))
	request.TopP = nil
	temperature := 1.0
	request.Temperature = &temperature
	return true
}

// ApplyPost46ThinkingSuffix converts the compatibility "-thinking" model
// suffix into adaptive high effort for Claude models that reject budgets.
func ApplyPost46ThinkingSuffix(request *dto.ClaudeRequest) bool {
	if request == nil || !strings.HasSuffix(request.Model, "-thinking") {
		return false
	}
	baseModel := strings.TrimSuffix(request.Model, "-thinking")
	if !reasoning.IsClaudePost46AdaptiveThinkingModel(baseModel) {
		return false
	}
	request.Model = baseModel
	return SetEffort(request, defaultEffort)
}

// SetEffort configures adaptive thinking for Claude models that reject manual
// thinking budgets and non-default sampling parameters.
func SetEffort(request *dto.ClaudeRequest, effort string) bool {
	if request == nil || !reasoning.IsClaudeEffortLevel(effort) {
		return false
	}
	request.Thinking = &dto.Thinking{
		Type:    "adaptive",
		Display: "summarized",
	}
	request.OutputConfig = json.RawMessage(fmt.Sprintf(`{"effort":"%s"}`, effort))
	clearSamplingParams(request)
	return true
}

// Normalize enforces the post-4.6 Claude adaptive-thinking request contract.
func Normalize(request *dto.ClaudeRequest) {
	if request == nil || !reasoning.IsClaudePost46AdaptiveThinkingModel(request.Model) {
		return
	}
	clearSamplingParams(request)
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
			request.OutputConfig = json.RawMessage(fmt.Sprintf(`{"effort":"%s"}`, defaultEffort))
		}
	case "adaptive":
		request.Thinking.BudgetTokens = nil
		if request.Thinking.Display == "" {
			request.Thinking.Display = "summarized"
		}
		if effort := request.GetEfforts(); effort != "" && !reasoning.IsClaudeEffortLevel(effort) {
			request.OutputConfig = json.RawMessage(fmt.Sprintf(`{"effort":"%s"}`, defaultEffort))
		}
	}
}

func clearSamplingParams(request *dto.ClaudeRequest) {
	request.Temperature = nil
	request.TopP = nil
	request.TopK = nil
}
