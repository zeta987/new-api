package reasoning

import (
	"strings"

	"github.com/samber/lo"
)

var EffortSuffixes = []string{"-max", "-xhigh", "-high", "-medium", "-low", "-minimal"}

var OpenAIEffortSuffixes = []string{"-high", "-minimal", "-low", "-medium", "-none", "-xhigh"}

var DeepSeekV4EffortSuffixes = []string{"-none", "-max"}

var gpt56Models = []string{"gpt-5.6-luna", "gpt-5.6-terra", "gpt-5.6-sol"}

// TrimEffortSuffix -> modelName level(low) exists
func TrimEffortSuffix(modelName string) (string, string, bool) {
	return TrimEffortSuffixWithSuffixes(modelName, EffortSuffixes)
}

func TrimEffortSuffixWithSuffixes(modelName string, suffixes []string) (string, string, bool) {
	suffix, found := lo.Find(suffixes, func(s string) bool {
		return strings.HasSuffix(modelName, s)
	})
	if !found {
		return modelName, "", false
	}
	return strings.TrimSuffix(modelName, suffix), strings.TrimPrefix(suffix, "-"), true
}

func ParseOpenAIReasoningEffortFromModelSuffix(modelName string) (string, string) {
	baseModel, mode, effort, ok := ParseOpenAIReasoningModelSuffix(modelName)
	if !ok || mode != "" {
		return "", modelName
	}
	return effort, baseModel
}

// ParseOpenAIReasoningModelSuffix parses supported OpenAI model suffixes.
// GPT-5.6 uses the grammar <base>[-<mode>][-<effort>], where mode must
// precede effort. Other OpenAI models keep the existing effort-only behavior.
func ParseOpenAIReasoningModelSuffix(modelName string) (baseModel string, mode string, effort string, ok bool) {
	if _, _, isGPT56 := splitGPT56Model(modelName); isGPT56 {
		return ParseGPT56ReasoningModelSuffix(modelName)
	}

	baseModel, effort, ok = TrimEffortSuffixWithSuffixes(modelName, OpenAIEffortSuffixes)
	if !ok {
		return modelName, "", "", false
	}
	return baseModel, "", effort, true
}

// GPT56ReasoningWildcardModel returns the channel wildcard key for a valid
// GPT-5.6 reasoning suffix. Invalid suffixes never receive wildcard access.
func GPT56ReasoningWildcardModel(modelName string) (string, bool) {
	baseModel, _, _, ok := ParseGPT56ReasoningModelSuffix(modelName)
	if !ok {
		return "", false
	}
	return baseModel + "-*", true
}

// IsGPT56ReasoningWildcard reports whether modelName is a configuration-only
// wildcard key. Clients must send a concrete validated suffix instead.
func IsGPT56ReasoningWildcard(modelName string) bool {
	for _, candidate := range gpt56Models {
		if modelName == candidate+"-*" {
			return true
		}
	}
	return false
}

// ParseGPT56ReasoningModelSuffix parses a validated GPT-5.6 mode and effort suffix.
func ParseGPT56ReasoningModelSuffix(modelName string) (baseModel string, mode string, effort string, ok bool) {
	baseModel, suffix, isGPT56 := splitGPT56Model(modelName)
	if !isGPT56 || suffix == "" {
		return modelName, "", "", false
	}

	parts := strings.Split(suffix, "-")
	switch len(parts) {
	case 1:
		if canonicalMode, validMode := canonicalGPT56ReasoningMode(parts[0]); validMode {
			return baseModel, canonicalMode, "", true
		}
		if isGPT56ReasoningEffort(parts[0]) {
			return baseModel, "", parts[0], true
		}
	case 2:
		canonicalMode, validMode := canonicalGPT56ReasoningMode(parts[0])
		if validMode && isGPT56ReasoningEffort(parts[1]) {
			return baseModel, canonicalMode, parts[1], true
		}
	}

	return modelName, "", "", false
}

func splitGPT56Model(modelName string) (baseModel string, suffix string, ok bool) {
	for _, candidate := range gpt56Models {
		if modelName == candidate {
			return candidate, "", true
		}
		prefix := candidate + "-"
		if strings.HasPrefix(modelName, prefix) {
			return candidate, strings.TrimPrefix(modelName, prefix), true
		}
	}
	return "", "", false
}

func canonicalGPT56ReasoningMode(mode string) (string, bool) {
	switch mode {
	case "pro":
		return "pro", true
	case "standard", "stanard":
		return "standard", true
	default:
		return "", false
	}
}

func isGPT56ReasoningEffort(effort string) bool {
	switch effort {
	case "none", "low", "medium", "high", "xhigh", "max":
		return true
	default:
		return false
	}
}

func ParseDeepSeekV4ThinkingSuffix(modelName string) (baseModel string, thinkingType string, effort string, ok bool) {
	baseModel, suffix, ok := TrimEffortSuffixWithSuffixes(modelName, DeepSeekV4EffortSuffixes)
	if !ok || !strings.HasPrefix(baseModel, "deepseek-v4-") {
		return modelName, "", "", false
	}
	switch suffix {
	case "none":
		return baseModel, "disabled", "", true
	case "max":
		return baseModel, "enabled", "max", true
	default:
		return modelName, "", "", false
	}
}
