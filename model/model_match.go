package model

import (
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/setting/reasoning"
)

// ModelMatchCandidates returns model keys in compatibility order. Exact model
// configuration wins, followed by a validated reasoning wildcard, the legacy
// formatted name, and the routing-normalized base.
func ModelMatchCandidates(modelName string) []string {
	if reasoning.IsOpenAIReasoningWildcard(modelName) {
		return nil
	}

	rawCandidates := []string{modelName}
	if wildcard, ok := reasoning.OpenAIReasoningWildcardModel(modelName); ok {
		rawCandidates = append(rawCandidates, wildcard)
	}
	rawCandidates = append(rawCandidates, ratio_setting.FormatMatchingModelName(modelName))
	rawCandidates = append(rawCandidates, ratio_setting.RoutingMatchModelName(modelName))

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

// PricingKeyForModel returns the key a price for modelName would actually be
// configured under, mirroring the order ModelPricingCandidates resolves in:
// the OpenAI reasoning families, then a plain effort suffix, then the wildcard
// and prefix normalization that still owns GLM and the thinking-budget
// aliases. Views that list models against configured prices must collapse
// through this, or a variant already covered by its base row looks unpriced.
//
// It lives here rather than in controller so callers reach ratio_setting
// through this package's existing dependency instead of importing it directly.
func PricingKeyForModel(modelName string) string {
	if base, known := reasoning.OpenAIReasoningBaseModel(modelName); known {
		return base
	}
	if suffixBase := reasoning.EffortSuffixBaseModelName(modelName); suffixBase != "" {
		return suffixBase
	}
	if formatted := ratio_setting.FormatMatchingModelName(modelName); formatted != "" {
		return formatted
	}
	return modelName
}
