package reasoning

import "strings"

const GPT6Astra = "gpt-6-astra"

// ParseGPT6AstraReasoningModelSuffix retains the GPT-5.6 alias grammar
// with Astra's effort vocabulary and canonical mode spellings.
func ParseGPT6AstraReasoningModelSuffix(model string) (string, string, string, bool) {
	if !strings.HasPrefix(model, GPT6Astra+"-") {
		return model, "", "", false
	}
	parts := strings.Split(strings.TrimPrefix(model, GPT6Astra+"-"), "-")
	mode, effort := "", ""
	if parts[0] == "standard" || parts[0] == "pro" {
		mode = parts[0]
		parts = parts[1:]
	}
	if len(parts) == 1 && IsGPT6AstraEffort(parts[0]) {
		effort = parts[0]
		parts = parts[1:]
	}
	if len(parts) != 0 || mode == "" && effort == "" {
		return model, "", "", false
	}
	return GPT6Astra, mode, effort, true
}

func IsGPT6AstraEffort(effort string) bool {
	switch effort {
	case "low", "medium", "high", "xhigh", "max":
		return true
	default:
		return false
	}
}

func IsGPT6AstraModel(model string) bool {
	if model == GPT6Astra {
		return true
	}
	_, _, _, ok := ParseGPT6AstraReasoningModelSuffix(model)
	return ok
}

func OpenAIReasoningWildcardModel(model string) (string, bool) {
	if base, _, _, ok := ParseGPT6AstraReasoningModelSuffix(model); ok {
		return base + "-*", true
	}
	return GPT56ReasoningWildcardModel(model)
}

func IsOpenAIReasoningWildcard(model string) bool {
	return model == GPT6Astra+"-*" || IsGPT56ReasoningWildcard(model)
}
