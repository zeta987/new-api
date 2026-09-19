package billing_setting

import (
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/pkg/jsplugin"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
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

// Effort variants resolve through the base model's candidates, so a base
// tiered_expr row prices them unless the variant carries its own billing mode.
// This pins the precedence resolveBillingConfig already has; it asserts current
// behavior and must not be used to justify changing that order.
func TestEffortSuffixBillingModePrecedence(t *testing.T) {
	originalModes := lo.Assign(billingSetting.BillingMode)
	originalExprs := lo.Assign(billingSetting.BillingExpr)
	t.Cleanup(func() {
		billingSetting.BillingMode = originalModes
		billingSetting.BillingExpr = originalExprs
	})

	billingSetting.BillingMode = map[string]string{
		"claude-fable-5":      BillingModeTieredExpr,
		"claude-fable-5-low":  BillingModeRatio,
		"kimi-k3":             BillingModeTieredExpr,
		"deepseek-v4-pro":     BillingModeTieredExpr,
		"deepseek-v4-pro-max": BillingModeRatio,
	}
	billingSetting.BillingExpr = map[string]string{
		"claude-fable-5":  "tier(\"fable\", p * 2)",
		"kimi-k3":         "tier(\"kimi\", p * 1)",
		"deepseek-v4-pro": "tier(\"deepseek\", p * 3)",
	}

	tests := []struct {
		model    string
		wantMode string
		wantExpr string
	}{
		// No billing_mode entry of its own: the base expression answers, even
		// though defaultModelRatio seeds a ratio row for the variant.
		{model: "claude-fable-5-high", wantMode: BillingModeTieredExpr, wantExpr: "tier(\"fable\", p * 2)"},
		{model: "claude-fable-5-max", wantMode: BillingModeTieredExpr, wantExpr: "tier(\"fable\", p * 2)"},
		{model: "claude-fable-5", wantMode: BillingModeTieredExpr, wantExpr: "tier(\"fable\", p * 2)"},
		// The variant carries its own mode, so candidate[0] wins. A row saved
		// through the admin page always writes its own billing_mode entry.
		{model: "claude-fable-5-low", wantMode: BillingModeRatio},
		{model: "deepseek-v4-pro-max", wantMode: BillingModeRatio},
		// Families wired up in this branch reach their base expression too.
		{model: "kimi-k3-high", wantMode: BillingModeTieredExpr, wantExpr: "tier(\"kimi\", p * 1)"},
		{model: "kimi-k3-none", wantMode: BillingModeTieredExpr, wantExpr: "tier(\"kimi\", p * 1)"},
		{model: "deepseek-v4-pro-high", wantMode: BillingModeTieredExpr, wantExpr: "tier(\"deepseek\", p * 3)"},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			assert.Equal(t, tt.wantMode, GetBillingMode(tt.model))
			expr, ok := GetBillingExpr(tt.model)
			if tt.wantExpr == "" {
				assert.False(t, ok)
				return
			}
			require.True(t, ok)
			assert.Equal(t, tt.wantExpr, expr)
		})
	}
}

func TestSmokeTestTaskExprValidatesDeclaredUsageVectors(t *testing.T) {
	videoSchema := map[string]jsplugin.UsageFieldSchema{
		"seconds": {Type: "number", Unit: "second"},
		"mode":    {Enum: []string{"std", "pro"}},
		"quality": {Enum: []string{"sd", "hd"}},
	}

	tests := []struct {
		name          string
		schema        map[string]jsplugin.UsageFieldSchema
		expression    string
		expectedError string
	}{
		{
			name:          "fixed prices are not task usage prices",
			schema:        videoSchema,
			expression:    `true ? tier("normal", u("seconds") * 0.4) : tier("fixed", fixed(0.01))`,
			expectedError: "fixed pricing is not supported for task usage expressions",
		},
		{
			name:       "declared numeric and enum facts",
			schema:     videoSchema,
			expression: `u("mode") == "pro" ? tier("pro", u("seconds") * 0.8) : tier("std", u("seconds") * 0.4)`,
		},
		{
			name:          "undeclared literal key",
			schema:        videoSchema,
			expression:    `tier("base", u("clips") * 0.1)`,
			expectedError: `usage key "clips" is not declared`,
		},
		{
			name:          "negative duration boundary",
			schema:        videoSchema,
			expression:    fmt.Sprintf(`u("seconds") == %d ? -1 : 0`, relaycommon.MaxTaskDurationSeconds),
			expectedError: "result must be finite and non-negative",
		},
		{
			name:          "negative count boundary",
			schema:        map[string]jsplugin.UsageFieldSchema{"clips": {Type: "number", Unit: "count"}},
			expression:    fmt.Sprintf(`u("clips") == %d ? -1 : 0`, dto.MaxImageN),
			expectedError: "result must be finite and non-negative",
		},
		{
			name:          "negative token boundary",
			schema:        map[string]jsplugin.UsageFieldSchema{"tokens": {Type: "number", Unit: "token"}},
			expression:    fmt.Sprintf(`u("tokens") == %d ? -1 : 0`, common.MaxQuota),
			expectedError: "result must be finite and non-negative",
		},
		{
			name:          "negative credit boundary",
			schema:        map[string]jsplugin.UsageFieldSchema{"units": {Type: "number", Unit: "credit"}},
			expression:    fmt.Sprintf(`u("units") == %d ? -1 : 0`, common.MaxQuota),
			expectedError: "result must be finite and non-negative",
		},
		{
			name:          "negative enum combination",
			schema:        videoSchema,
			expression:    `u("mode") == "pro" && u("quality") == "hd" ? -1 : 0`,
			expectedError: "result must be finite and non-negative",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			err := SmokeTestTaskExpr(testCase.expression, testCase.schema)
			if testCase.expectedError == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, testCase.expectedError)
		})
	}
}

func TestSmokeTestTaskExprCapsOversizedEnumProductsAtLastCombination(t *testing.T) {
	schema := make(map[string]jsplugin.UsageFieldSchema, 7)
	condition := ""
	for index := range 7 {
		schema[fmt.Sprintf("enum_%d", index)] = jsplugin.UsageFieldSchema{Enum: []string{"first", "middle", "last"}}
		if condition != "" {
			condition += " && "
		}
		condition += fmt.Sprintf(`u("enum_%d") == "last"`, index)
	}

	err := SmokeTestTaskExpr(condition+" ? -1 : 0", schema)
	require.ErrorContains(t, err, "result must be finite and non-negative")
}

func TestSmokeTestExprRejectsTaskUsageWithoutSchema(t *testing.T) {
	err := SmokeTestExpr(`u("mode") == "std" ? 1 : 2`)
	require.Error(t, err)
	assert.ErrorContains(t, err, "mode")
	assert.ErrorContains(t, err, "no task plugin usage schema")

	require.NoError(t, SmokeTestExpr(`tier("base", p * 2 + c * 8)`))
}
