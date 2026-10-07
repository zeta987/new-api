package billing_setting

// Built-in token prices use actual USD per million tokens. Keep new model
// defaults here instead of splitting them across the legacy ratio tables.
var builtinBillingExpr = map[string]string{
	// https://developers.openai.com/api/docs/pricing (Standard, 2026-09-09).
	// The Images API reports image output in output_tokens, normalized to c.
	"gpt-image-2":            `tier("standard", p * 5 + cr * 1.25 + img * 8 + img_cr * 2 + c * 30)`,
	"gpt-image-2.5-sunburst": `tier("standard", p * 5 + cr * 1.25 + img * 8 + img_cr * 2 + c * 30)`,
	"gpt-image-2.5-flare":    `tier("standard", p * 5 + cr * 1.25 + img * 8 + img_cr * 2 + c * 30)`,
	// https://developers.openai.com/api/docs/models/gpt-6-astra
	// Standard pricing; the long-context rates apply to the whole request.
	// Do not infer service-tier discounts from incoming request parameters:
	// channels filter service_tier by default, so it may not reach the upstream.
	"gpt-6-astra": `len <= 272000 ? tier("standard", p * 10 + c * 50 + cr * 1 + cc * 12.5) : tier("long_context", p * 20 + c * 75 + cr * 2 + cc * 25)`,
	// https://developers.openai.com/api/docs/models/gpt-6-sol
	// https://developers.openai.com/api/docs/models/gpt-6-luna
	"gpt-6-sol":  `len <= 272000 ? tier("standard", p * 2 + c * 10 + cr * 0.2 + cc * 2.5) : tier("long_context", p * 4 + c * 15 + cr * 0.4 + cc * 5)`,
	"gpt-6-luna": `len <= 272000 ? tier("standard", p * 0.1 + c * 0.5 + cr * 0.01 + cc * 0.125) : tier("long_context", p * 0.2 + c * 0.75 + cr * 0.02 + cc * 0.25)`,
	// https://platform.claude.com/docs/en/models/opus-5-5/overview
	// Claude's 1M context uses the standard token rates throughout.
	"claude-opus-5-5": `tier("standard", p * 4 + c * 20 + cr * 0.2 + cc * 5 + cc1h * 8)`,
	// https://platform.claude.com/docs/en/models/sonnet-5-5/overview
	// Cache reads dropped to 0.05x on 2026-10-07 (release notes).
	"claude-sonnet-5-5": `tier("standard", p * 2 + c * 10 + cr * 0.1 + cc * 2.5 + cc1h * 4)`,
	// https://platform.claude.com/docs/en/about-claude/pricing
	// The $2 / $10 launch price became Sonnet 5's standard price.
	"claude-sonnet-5": `tier("standard", p * 2 + c * 10 + cr * 0.2 + cc * 2.5 + cc1h * 4)`,
	// https://platform.claude.com/docs/en/models/fable-5-1/overview
	// Cache reads are 0.025x the base input price; Mythos 5.1 is priced the same.
	"claude-fable-5-1":  `tier("standard", p * 10 + c * 50 + cr * 0.25 + cc * 12.5 + cc1h * 20)`,
	"claude-mythos-5-1": `tier("standard", p * 10 + c * 50 + cr * 0.25 + cc * 12.5 + cc1h * 20)`,
	// https://platform.claude.com/docs/en/models/fable-5/overview
	// Mythos 5 is priced the same as Fable 5.
	"claude-fable-5":  `tier("standard", p * 10 + c * 50 + cr * 1 + cc * 12.5 + cc1h * 20)`,
	"claude-mythos-5": `tier("standard", p * 10 + c * 50 + cr * 1 + cc * 12.5 + cc1h * 20)`,
	// https://platform.claude.com/docs/en/about-claude/pricing
	// Prompts over 100,000 tokens move every Haiku 5.5 rate to the higher card.
	"claude-haiku-5-5": `len <= 100000 ? tier("standard", p * 0.1 + c * 0.5 + cr * 0.01 + cc * 0.125 + cc1h * 0.2) : tier("long_context", p * 0.5 + c * 2.5 + cr * 0.05 + cc * 0.625 + cc1h * 1)`,
}
