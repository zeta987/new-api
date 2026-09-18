package reasoning

import (
	kitreasoning "github.com/QuantumNous/new-api/relaykit/relayconvert/reasoning"
	"github.com/QuantumNous/new-api/setting/model_setting"
)

// EffortSuffixBaseModelName returns the base model name of a plain
// effort-suffixed alias, so a single pricing row per base model can serve every
// effort variant of that model. It returns "" when nothing was stripped.
//
// Unlike BaseModelName it never touches a name that carries explicit @
// modifiers: those resolve through the canonical @effort:/@thinking: billing
// ladder, and collapsing them to the base here would make every pricing getter
// report a canonical candidate as configured purely because the base row
// exists, silently disabling per-effort canonical pricing.
//
// The thinking-suffix blacklist is checked on both the incoming name and the
// stripped base, mirroring BaseModelName, so it stays the single per-name
// escape hatch that keeps a model identity opaque.
func EffortSuffixBaseModelName(modelName string) string {
	if modelName == "" || kitreasoning.ParseModelModifiers(modelName).HasModifiers() {
		return ""
	}
	if model_setting.ShouldPreserveThinkingSuffix(modelName) {
		return ""
	}

	base, _, found, err := ParseLegacyModelSuffix(
		modelName,
		model_setting.GetClaudeSettings().ThinkingAdapterEnabled,
		model_setting.GetGeminiSettings().ThinkingAdapterEnabled,
	)
	if err != nil || !found {
		deepSeekBase, _, _, isDeepSeek := kitreasoning.ParseDeepSeekV4ThinkingSuffix(modelName)
		kimiBase, _, isKimi := kitreasoning.ParseKimiReasoningEffortSuffix(modelName)
		grokBase, _, isGrok := kitreasoning.ParseGrokReasoningEffortSuffix(modelName)
		switch {
		case isDeepSeek:
			base = deepSeekBase
		case isKimi:
			base = kimiBase
		case isGrok:
			base = grokBase
		default:
			return ""
		}
	}

	if base == "" || base == modelName || model_setting.ShouldPreserveThinkingSuffix(base) {
		return ""
	}
	return base
}
