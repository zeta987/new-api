package reasoning

import (
	"strconv"
	"strings"
)

// ParseGrokReasoningEffortSuffix splits a grok effort alias into its base model
// and effort. Standard grok-4.<minor> names support low, medium and high;
// grok-4.6 and later also support xhigh. Later major versions retain the
// released full vocabulary.
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
	if !ok || major < 4 {
		return modelName, "", false
	}
	if effort == "xhigh" && major == 4 && minor < 6 {
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
