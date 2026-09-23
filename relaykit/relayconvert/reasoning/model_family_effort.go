package reasoning

import "regexp"

var (
	claudeEffortLevels      = []string{"low", "medium", "high", "xhigh", "max"}
	geminiFullEffortLevels  = []string{"minimal", "low", "medium", "high"}
	geminiCoreEffortLevels  = []string{"low", "medium", "high"}
	glmEffortLevels         = []string{"low", "high", "max"}
	toggleEffortLevels      = []string{"none", "low", "high", "max"}
	grokCoreEffortLevels    = []string{"low", "medium", "high"}
	grokEffortLevels        = []string{"low", "medium", "high", "xhigh"}
	standardClaudeFiveModel = regexp.MustCompile(`^claude-(?:fable|opus|sonnet)-5(?:-[1-9][0-9]?)?$`)
)

// effortSuffixVocabulary is the authoritative per-base effort vocabulary used
// to expose effort variants of a base model that a channel registers on its
// own.
//
// The table and positive family rules must not be derived from the renderer
// capability tables (claudeCapabilitiesFor, geminiLevelForEffort). Those
// describe what the upstream API tolerates, which is wider than what is
// published here: they would, for example, also admit "minimal" for
// gemini-3.7-flash and gemini-3.8-flash.
//
// Claude 5.x models deliberately omit "none" even though some of their
// capabilities report supportsDisable and the official API accepts a disabled
// thinking block. With thinking disabled those models intermittently write
// tool calls into visible assistant text instead of emitting tool_use blocks,
// which does not raise an error and silently corrupts agent loops, and they
// leak <thinking> tags into the reply. Low effort is the documented
// alternative, so only the effort levels are published.
var effortSuffixVocabulary = map[string][]string{
	"gemini-3.1-flash-lite":  geminiFullEffortLevels,
	"gemini-3.5-flash-lite":  geminiFullEffortLevels,
	"gemini-3.5-flash":       geminiFullEffortLevels,
	"gemini-3.6-flash":       geminiFullEffortLevels,
	"gemini-3.7-flash":       geminiCoreEffortLevels,
	"gemini-3.8-flash":       geminiCoreEffortLevels,
	"gemini-3.1-pro-preview": geminiCoreEffortLevels,

	// GLM publishes the 5.3 line only; 5.2 is deliberately absent.
	"glm-5.3":        glmEffortLevels,
	"glm-5.3-flash":  glmEffortLevels,
	"glm-5.3-flashx": glmEffortLevels,

	"deepseek-flash":  toggleEffortLevels,
	"deepseek-v4-pro": toggleEffortLevels,

	"kimi-k3": toggleEffortLevels,
}

// EffortSuffixModelNames returns the base model followed by each published
// effort variant, or nil when the name is not a known base. A namespace prefix
// such as "vendor/" is carried onto every returned name.
func EffortSuffixModelNames(base string) []string {
	modelName := lastModelPathSegment(base)
	levels, known := effortSuffixVocabulary[modelName]
	if IsStandardClaudeFiveModel(modelName) {
		levels = claudeEffortLevels
		known = true
	}
	if major, minor, standardGrok := parseStandardGrokVersion(modelName); standardGrok && major == 4 {
		levels = grokCoreEffortLevels
		if minor >= 6 {
			levels = grokEffortLevels
		}
		known = true
	}
	if !known {
		return nil
	}
	names := make([]string, 0, len(levels)+1)
	names = append(names, base)
	for _, level := range levels {
		names = append(names, base+"-"+level)
	}
	return names
}

func IsStandardClaudeFiveModel(name string) bool {
	return standardClaudeFiveModel.MatchString(lastModelPathSegment(name))
}
