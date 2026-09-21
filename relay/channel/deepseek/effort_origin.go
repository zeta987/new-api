package deepseek

import (
	"strings"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/reasoning"
)

// resolveDeepSeekV4Suffix reads the thinking suffix from the upstream model
// name and falls back to the origin name when a model mapping already replaced
// the suffixed alias with the model's base name or a pinned snapshot of it.
//
// The returned model name is the one the request must carry upstream, so a
// mapping target such as deepseek-v4-pro-0501 is never rewritten back to
// deepseek-v4-pro. The origin fallback only applies when the upstream name
// still belongs to the same model, i.e. it starts with the origin's base, so an
// unrelated mapping target never inherits another model's effort.
func resolveDeepSeekV4Suffix(info *relaycommon.RelayInfo, modelName string) (baseModel string, thinkingType string, effort string, ok bool) {
	if baseModel, thinkingType, effort, ok = reasoning.ParseDeepSeekV4ThinkingSuffix(modelName); ok {
		return baseModel, thinkingType, effort, true
	}
	if info == nil {
		return modelName, "", "", false
	}
	originBase, thinkingType, effort, ok := reasoning.ParseDeepSeekV4ThinkingSuffix(info.OriginModelName)
	if !ok || !strings.HasPrefix(modelName, originBase) {
		return modelName, "", "", false
	}
	return modelName, thinkingType, effort, true
}
