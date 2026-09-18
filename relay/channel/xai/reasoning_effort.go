package xai

import (
	"strconv"
	"strings"
)

func parseGrokReasoningEffortSuffix(modelName string) (string, string, bool) {
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
