package ratio_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatMatchingModelNameNormalizesExactGLM52Aliases(t *testing.T) {
	for _, model := range []string{"glm-5.2-none", "glm-5.2-high", "glm-5.2-max"} {
		assert.Equal(t, "glm-5.2", FormatMatchingModelName(model))
	}
	for _, model := range []string{"glm-5.2", "glm-5.2-low", "glm-5.2-max-extra", "glm-5.1-high"} {
		assert.Equal(t, model, FormatMatchingModelName(model))
	}
}
