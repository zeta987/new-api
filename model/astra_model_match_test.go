package model

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAstraModelMatchCandidates(t *testing.T) {
	assert.Equal(t, []string{"gpt-6-astra-pro-max", "gpt-6-astra-*", "gpt-6-astra"}, ModelMatchCandidates("gpt-6-astra-pro-max"))
	assert.Empty(t, ModelMatchCandidates("gpt-6-astra-*"))
	for _, invalid := range []string{"gpt-6-astra-none", "gpt-6-astra-minimal", "gpt-6-astra-high-pro", "gpt-6-astra-pro-ultra", "gpt-6-astra-stanard-high"} {
		assert.Equal(t, []string{invalid}, ModelMatchCandidates(invalid))
	}
}
