package service

import (
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/service/claudeadaptive"
)

func SetClaudeAdaptiveEffort(request *dto.ClaudeRequest, effort string) bool {
	return claudeadaptive.SetEffort(request, effort)
}

func NormalizeClaudePost46AdaptiveRequest(request *dto.ClaudeRequest) {
	claudeadaptive.Normalize(request)
}
