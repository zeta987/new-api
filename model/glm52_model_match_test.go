package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestModelMatchCandidatesIncludeGLM52BaseAfterExactAlias(t *testing.T) {
	for _, alias := range []string{"glm-5.2-none", "glm-5.2-high", "glm-5.2-max"} {
		assert.Equal(t, []string{alias, "glm-5.2"}, ModelMatchCandidates(alias))
	}
	assert.Equal(t, []string{"glm-5.2-low"}, ModelMatchCandidates("glm-5.2-low"))
}
