package model

import (
	"testing"

	"github.com/QuantumNous/new-api/setting/reasoning"
	"github.com/stretchr/testify/assert"
)

func TestModelMatchCandidatesIncludeGenericGLMBaseAfterExactAlias(t *testing.T) {
	for _, alias := range []string{"glm-5.3-flash-low", "glm-5.3-flash-high", "glm-5.3-flash-max", "glm-future-model-xhigh"} {
		base, _, ok := reasoning.ParseGLMReasoningEffortSuffix(alias)
		assert.True(t, ok)
		assert.Equal(t, []string{alias, base}, ModelMatchCandidates(alias))
	}
	assert.Equal(t, []string{"glm-low"}, ModelMatchCandidates("glm-low"))
}
