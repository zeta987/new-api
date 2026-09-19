package reasoning

import (
	"strconv"
	"strings"
)

// ParseGrokReasoningEffortSuffix splits a grok effort alias into its base model
// and effort. Only standard grok-<major>.<minor> names from 4.5 onwards carry
// effort aliases, and 4.5 itself does not accept xhigh.
//
// It lives here rather than in the xAI adaptor so pricing, routing and model
// list expansion share one vocabulary.
func ParseGrokReasoningEffortSuffix(modelName string) (string, string, bool) {
	separatorIndex := strings.LastIndex(modelName, "-")
	if separatorIndex < 0 {
		return modelName, "", false
	}

	baseModel := modelName[:separatorIndex]
	effort := modelName[separatorIndex+1:]
	switch effort {
	case "low", "medium", "high", "xhigh":
	default:
		return modelName, "", false
	}

	major, minor, ok := parseStandardGrokVersion(baseModel)
	if !ok || major < 4 || (major == 4 && minor < 5) {
		return modelName, "", false
	}
	if effort == "xhigh" && major == 4 && minor == 5 {
		return modelName, "", false
	}
	return baseModel, effort, true
}

// IsStandardGrokModel reports whether modelName is a bare grok-<major>.<minor>
// model name, i.e. one that carries no effort alias of its own.
func IsStandardGrokModel(modelName string) bool {
	_, _, ok := parseStandardGrokVersion(modelName)
	return ok
}

func parseStandardGrokVersion(modelName string) (int, int, bool) {
	version, ok := strings.CutPrefix(modelName, "grok-")
	if !ok {
		return 0, 0, false
	}

	parts := strings.Split(version, ".")
	if len(parts) != 2 {
		return 0, 0, false
	}
	major, majorErr := strconv.Atoi(parts[0])
	minor, minorErr := strconv.Atoi(parts[1])
	if majorErr != nil || minorErr != nil || major < 0 || minor < 0 {
		return 0, 0, false
	}
	if strconv.Itoa(major) != parts[0] || strconv.Itoa(minor) != parts[1] {
		return 0, 0, false
	}
	return major, minor, true
}
