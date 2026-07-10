package billing_setting

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGPT56TieredBillingWildcardUsesOneCandidate(t *testing.T) {
	originalModes := lo.Assign(billingSetting.BillingMode)
	originalExprs := lo.Assign(billingSetting.BillingExpr)
	t.Cleanup(func() {
		billingSetting.BillingMode = originalModes
		billingSetting.BillingExpr = originalExprs
	})

	billingSetting.BillingMode = map[string]string{
		"gpt-5.6-*": BillingModeTieredExpr,
	}
	billingSetting.BillingExpr = map[string]string{
		"gpt-5.6-*": "p + c",
	}

	assert.Equal(t, BillingModeTieredExpr, GetBillingMode("gpt-5.6-luna-pro-max"))
	expr, ok := GetBillingExpr("gpt-5.6-luna-pro-max")
	require.True(t, ok)
	assert.Equal(t, "p + c", expr)
}

func TestGPT56TieredBillingDoesNotMixCandidateKeys(t *testing.T) {
	originalModes := lo.Assign(billingSetting.BillingMode)
	originalExprs := lo.Assign(billingSetting.BillingExpr)
	t.Cleanup(func() {
		billingSetting.BillingMode = originalModes
		billingSetting.BillingExpr = originalExprs
	})

	billingSetting.BillingMode = map[string]string{
		"gpt-5.6-luna-*": BillingModeTieredExpr,
	}
	billingSetting.BillingExpr = map[string]string{
		"gpt-5.6-*": "p + c",
	}

	assert.Equal(t, BillingModeTieredExpr, GetBillingMode("gpt-5.6-luna-pro-max"))
	_, ok := GetBillingExpr("gpt-5.6-luna-pro-max")
	assert.False(t, ok)
}
