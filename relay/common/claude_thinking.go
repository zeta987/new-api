package common

import (
	"github.com/QuantumNous/new-api/dto"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// NormalizeClaudeThinkingSampling removes top_p whenever a Claude request
// enters thinking/adaptive mode.
func NormalizeClaudeThinkingSampling(request *dto.ClaudeRequest) {
	if request == nil || request.Thinking == nil {
		return
	}
	request.TopP = nil
}

// RemoveClaudeTopPWhenThinkingJSON strips top_p from Claude payloads that
// already contain a thinking config, including payloads modified by overrides.
func RemoveClaudeTopPWhenThinkingJSON(jsonData []byte) ([]byte, error) {
	if len(jsonData) == 0 {
		return jsonData, nil
	}

	thinking := gjson.GetBytes(jsonData, "thinking")
	if !thinking.Exists() || thinking.Type == gjson.Null {
		return jsonData, nil
	}

	if !gjson.GetBytes(jsonData, "top_p").Exists() {
		return jsonData, nil
	}

	return sjson.DeleteBytes(jsonData, "top_p")
}
