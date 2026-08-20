package constant

// GLMReasoningEffortAliasChannelTypes lists the channel types whose adaptors
// translate a validated GLM reasoning effort alias into an upstream reasoning
// effort. A channel type outside this list never serves such an alias.
var GLMReasoningEffortAliasChannelTypes = []int{
	ChannelTypeZhipu_v4,
	ChannelTypeOpenAI,
	ChannelTypeOpenRouter,
}

// SupportsGLMReasoningEffortAlias reports whether channelType can serve a
// validated GLM reasoning effort alias.
func SupportsGLMReasoningEffortAlias(channelType int) bool {
	for _, candidate := range GLMReasoningEffortAliasChannelTypes {
		if candidate == channelType {
			return true
		}
	}
	return false
}
