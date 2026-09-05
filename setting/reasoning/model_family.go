package reasoning

import (
	kitreasoning "github.com/QuantumNous/new-api/relaykit/relayconvert/reasoning"
	"github.com/QuantumNous/new-api/setting/model_setting"
)

func OpenAIReasoningBaseModel(name string) (string, bool) {
	base, known := kitreasoning.OpenAIReasoningBaseModel(name)
	if !known || model_setting.ShouldPreserveThinkingSuffix(name) || model_setting.ShouldPreserveThinkingSuffix(base) {
		return name, false
	}
	return base, true
}

// ExpandOpenAIReasoningModels is shared by API discovery and model selectors.
// Existing exact aliases and unrelated model names are retained in order.
func ExpandOpenAIReasoningModels(names []string) []string {
	result := make([]string, 0, len(names))
	seen := make(map[string]bool)
	for _, name := range names {
		variants := []string{name}
		if _, known := OpenAIReasoningBaseModel(name); known {
			variants = kitreasoning.OpenAIReasoningModelNames(name)
		} else if kitreasoning.IsOpenAIReasoningWildcard(name) {
			continue
		}
		for _, variant := range variants {
			if seen[variant] || (model_setting.ShouldPreserveThinkingSuffix(variant) && variant != name) {
				continue
			}
			seen[variant] = true
			result = append(result, variant)
		}
	}
	return result
}
