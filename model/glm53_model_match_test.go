package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestModelMatchCandidatesIncludeGLM53BaseAfterExactAlias(t *testing.T) {
	for _, alias := range []string{"glm-5.3-low", "glm-5.3-high", "glm-5.3-max"} {
		assert.Equal(t, []string{alias, "glm-5.3"}, ModelMatchCandidates(alias))
	}
	assert.Equal(t, []string{"glm-5.3-none"}, ModelMatchCandidates("glm-5.3-none"))
}
