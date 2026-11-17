package reasoning

import (
	"strconv"
	"strings"
)

// IsQwen4MaxModel matches the base ID, including within a provider namespace.
func IsQwen4MaxModel(modelName string) bool {
	_, bare := splitModelNamespace(modelName)
	return bare == "qwen4-max"
}

// IsQwenReasoningModel reports whether modelName is a supported Qwen
// reasoning base model. An opaque provider namespace is preserved.
func IsQwenReasoningModel(modelName string) bool {
	if IsQwen4MaxModel(modelName) {
		return true
	}
	_, bare := splitModelNamespace(modelName)
	if !strings.HasPrefix(bare, "qwen") {
		return false
	}

	version, familyAndSnapshot, found := strings.Cut(strings.TrimPrefix(bare, "qwen"), "-")
	if !found {
		return false
	}
	majorText, minorText, found := strings.Cut(version, ".")
	if !found || strings.Contains(minorText, ".") {
		return false
	}
	major, majorOK := parseQwenVersionSegment(majorText)
	minor, minorOK := parseQwenVersionSegment(minorText)
	if !majorOK || !minorOK || major < 3 || major == 3 && minor < 8 {
		return false
	}

	family, snapshot, hasSnapshot := strings.Cut(familyAndSnapshot, "-")
	if family != "max" && family != "flash" {
		return false
	}
	if !hasSnapshot {
		return true
	}
	return isQwenSnapshot(snapshot)
}

// ParseQwenReasoningEffortSuffix recognizes the effort aliases of supported
// Qwen models and canonicalizes the equivalent Qwen4-Max levels.
func ParseQwenReasoningEffortSuffix(modelName string) (baseModel string, effort string, ok bool) {
	lastHyphen := strings.LastIndexByte(modelName, '-')
	if lastHyphen < 0 {
		return modelName, "", false
	}
	baseModel = modelName[:lastHyphen]
	effort = modelName[lastHyphen+1:]
	if IsQwen4MaxModel(baseModel) {
		switch effort {
		case "none", "low", "high", "xhigh":
			return baseModel, effort, true
		case "minimal":
			return baseModel, "low", true
		case "medium":
			return baseModel, "high", true
		case "max":
			return baseModel, "xhigh", true
		default:
			return modelName, "", false
		}
	}
	if !isQwenReasoningEffort(effort) || !IsQwenReasoningModel(baseModel) {
		return modelName, "", false
	}
	return baseModel, effort, true
}

func parseQwenVersionSegment(segment string) (int, bool) {
	if segment == "" || len(segment) > 1 && segment[0] == '0' {
		return 0, false
	}
	for _, char := range segment {
		if char < '0' || char > '9' {
			return 0, false
		}
	}
	value, err := strconv.Atoi(segment)
	return value, err == nil
}

func isQwenSnapshot(snapshot string) bool {
	if len(snapshot) == 4 {
		return isQwenDigits(snapshot)
	}
	return len(snapshot) == 10 && snapshot[4] == '-' && snapshot[7] == '-' &&
		isQwenDigits(snapshot[:4]) && isQwenDigits(snapshot[5:7]) && isQwenDigits(snapshot[8:])
}

func isQwenDigits(value string) bool {
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return value != ""
}

func isQwenReasoningEffort(effort string) bool {
	switch effort {
	case "none", "low", "medium", "xhigh":
		return true
	default:
		return false
	}
}
