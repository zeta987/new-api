package reasoning

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDeepSeekV4ThinkingSuffixSupportsLow(t *testing.T) {
	for _, model := range []string{"deepseek-v4-flash-low", "deepseek-v4-pro-low"} {
		t.Run(model, func(t *testing.T) {
			baseModel, thinkingType, effort, ok := ParseDeepSeekV4ThinkingSuffix(model)

			require.True(t, ok)
			assert.Equal(t, model[:len(model)-len("-low")], baseModel)
			assert.Equal(t, "enabled", thinkingType)
			assert.Equal(t, "low", effort)
		})
	}
}
