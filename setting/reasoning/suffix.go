// Package reasoning re-exports the pure model-name effort-suffix helpers,
// which moved to the conversion kit (service/relayconvert/reasoning) as part
// of the relaykit extraction. Host code keeps importing this path unchanged.
package reasoning

import kitreasoning "github.com/QuantumNous/new-api/relaykit/relayconvert/reasoning"

var (
	EffortSuffixes           = kitreasoning.EffortSuffixes
	OpenAIEffortSuffixes     = kitreasoning.OpenAIEffortSuffixes
	DeepSeekV4EffortSuffixes = kitreasoning.DeepSeekV4EffortSuffixes
)

var (
	IsClaudeEffortLevel                       = kitreasoning.IsClaudeEffortLevel
	TrimEffortSuffix                          = kitreasoning.TrimEffortSuffix
	IsClaudeAdaptiveThinkingModel             = kitreasoning.IsClaudeAdaptiveThinkingModel
	IsClaudePost46AdaptiveThinkingModel       = kitreasoning.IsClaudePost46AdaptiveThinkingModel
	TrimEffortSuffixWithSuffixes              = kitreasoning.TrimEffortSuffixWithSuffixes
	ParseOpenAIReasoningEffortFromModelSuffix = kitreasoning.ParseOpenAIReasoningEffortFromModelSuffix
	ParseOpenAIReasoningModelSuffix           = kitreasoning.ParseOpenAIReasoningModelSuffix
	GPT56ReasoningWildcardModel               = kitreasoning.GPT56ReasoningWildcardModel
	IsGPT56ReasoningWildcard                  = kitreasoning.IsGPT56ReasoningWildcard
	ParseGPT56ReasoningModelSuffix            = kitreasoning.ParseGPT56ReasoningModelSuffix
	ParseDeepSeekV4ThinkingSuffix             = kitreasoning.ParseDeepSeekV4ThinkingSuffix
)
