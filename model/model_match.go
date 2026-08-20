package model

import (
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/setting/reasoning"
)

// ModelMatchCandidates returns model keys in compatibility order. Exact model
// configuration wins, followed by a validated GPT-5.6 reasoning wildcard and
// the normalized billing model.
func ModelMatchCandidates(modelName string) []string {
	if reasoning.IsGPT56ReasoningWildcard(modelName) {
		return nil
	}

	rawCandidates := []string{modelName}
	if wildcard, ok := reasoning.GPT56ReasoningWildcardModel(modelName); ok {
		rawCandidates = append(rawCandidates, wildcard)
	}
	rawCandidates = append(rawCandidates, ratio_setting.FormatMatchingModelName(modelName))

	candidates := make([]string, 0, len(rawCandidates))
	for _, candidate := range rawCandidates {
		if candidate == "" {
			continue
		}
		duplicate := false
		for _, existing := range candidates {
			if existing == candidate {
				duplicate = true
				break
			}
		}
		if !duplicate {
			candidates = append(candidates, candidate)
		}
	}
	return candidates
}

// requiresGLMEffortChannel reports whether modelName is a validated GLM
// reasoning effort alias, which only channel types that translate the alias
// into an upstream reasoning effort may serve.
func requiresGLMEffortChannel(modelName string) bool {
	_, _, ok := reasoning.ParseGLMReasoningEffortSuffix(modelName)
	return ok
}
