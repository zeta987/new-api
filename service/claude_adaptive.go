package service

import (
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/relayconvert/claudeadaptive"
)

func SetClaudeAdaptiveEffort(request *dto.ClaudeRequest, effort string) bool {
	return claudeadaptive.SetEffort(request, effort)
}

func NormalizeClaudePost46AdaptiveRequest(request *dto.ClaudeRequest) {
	claudeadaptive.Normalize(request)
}
