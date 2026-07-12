package reasoning

import (
	"strconv"
	"strings"

	"github.com/samber/lo"
)

var EffortSuffixes = []string{"-max", "-xhigh", "-high", "-medium", "-low", "-minimal"}

var OpenAIEffortSuffixes = []string{"-high", "-minimal", "-low", "-medium", "-none", "-xhigh"}

var DeepSeekV4EffortSuffixes = []string{"-none", "-max"}

func IsClaudeEffortLevel(effort string) bool {
	switch effort {
	case "low", "medium", "high", "xhigh", "max":
		return true
	default:
		return false
	}
}

// TrimEffortSuffix -> modelName level(low) exists
func TrimEffortSuffix(modelName string) (string, string, bool) {
	return TrimEffortSuffixWithSuffixes(modelName, EffortSuffixes)
}

func IsClaudeAdaptiveThinkingModel(modelName string) bool {
	major, minor, ok := parseClaudeModelVersion(modelName)
	if !ok {
		return false
	}
	return major > 4 || (major == 4 && minor >= 6)
}

func IsClaudePost46AdaptiveThinkingModel(modelName string) bool {
	major, minor, ok := parseClaudeModelVersion(modelName)
	if !ok {
		return false
	}
	return major > 4 || (major == 4 && minor > 6)
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

func parseClaudeModelVersion(modelName string) (int, int, bool) {
	if baseModel, _, ok := TrimEffortSuffix(modelName); ok {
		modelName = baseModel
	}
	modelName = strings.TrimSuffix(modelName, "-thinking")
	parts := strings.Split(modelName, "-")
	if len(parts) < 2 || parts[0] != "claude" {
		return 0, 0, false
	}

	for i := 1; i < len(parts); i++ {
		major, ok := parseClaudeVersionSegment(parts[i])
		if !ok {
			continue
		}

		minor := 0
		if i+1 < len(parts) {
			if parsedMinor, ok := parseClaudeVersionSegment(parts[i+1]); ok {
				minor = parsedMinor
			}
		}
		return major, minor, true
	}

	return 0, 0, false
}

func parseClaudeVersionSegment(part string) (int, bool) {
	if part == "" || len(part) > 2 {
		return 0, false
	}
	for _, char := range part {
		if char < '0' || char > '9' {
			return 0, false
		}
	}
	value, err := strconv.Atoi(part)
	return value, err == nil
}

func ParseOpenAIReasoningEffortFromModelSuffix(modelName string) (string, string) {
	baseModel, effort, ok := TrimEffortSuffixWithSuffixes(modelName, OpenAIEffortSuffixes)
	if !ok {
		return "", modelName
	}
	return effort, baseModel
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
