package reasoning

import "strings"

const GPT6Astra = "gpt-6-astra"

// ParseGPT6AstraReasoningModelSuffix parses aliases for the original GPT-6 Astra.
func ParseGPT6AstraReasoningModelSuffix(model string) (string, string, string, bool) {
	if strings.HasPrefix(model, GPT6Astra+"-") {
		return ParseGPT6RequiredReasoningModelSuffix(model)
	}
	return model, "", "", false
}

func splitGPT6RequiredReasoningModel(model string) (base, suffix string, known bool) {
	for _, candidate := range [...]string{GPT6Astra, "gpt-6.2-astra", "gpt-6.2-sol"} {
		if model == candidate {
			return candidate, "", true
		}
		if tail, ok := strings.CutPrefix(model, candidate+"-"); ok {
			return candidate, tail, true
		}
	}
	return "", "", false
}

// ParseGPT6RequiredReasoningModelSuffix recognizes mode and effort aliases for
// GPT-6 Astra and GPT-6.2 models, which require reasoning.
func ParseGPT6RequiredReasoningModelSuffix(model string) (string, string, string, bool) {
	base, suffix, known := splitGPT6RequiredReasoningModel(model)
	if !known || suffix == "" {
		return model, "", "", false
	}
	parts := strings.Split(suffix, "-")
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
	return base, mode, effort, true
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

// IsGPT6RequiredReasoningModel matches the base names and their valid aliases.
func IsGPT6RequiredReasoningModel(model string) bool {
	base, _, _ := splitGPT6RequiredReasoningModel(model)
	if base != "" && model == base {
		return true
	}
	_, _, _, ok := ParseGPT6RequiredReasoningModelSuffix(model)
	return ok
}

func OpenAIReasoningWildcardModel(model string) (string, bool) {
	if base, _, _, ok := ParseGPT6RequiredReasoningModelSuffix(model); ok {
		return base + "-*", true
	}
	if base, _, _, ok := ParseStandardGPT6ReasoningModelSuffix(model); ok {
		return base + "-*", true
	}
	return GPT56ReasoningWildcardModel(model)
}

func IsOpenAIReasoningWildcard(model string) bool {
	_, strictSuffix, strict := splitGPT6RequiredReasoningModel(model)
	if (strict && strictSuffix == "*") || IsGPT56ReasoningWildcard(model) {
		return true
	}
	_, suffix, standard := splitStandardGPT6Model(model)
	return standard && suffix == "*"
}
