package reasoning

// OpenAIReasoningBaseModel recognizes only registered model families and their
// validated aliases. Unrecognized vendor IDs remain opaque.
func OpenAIReasoningBaseModel(name string) (string, bool) {
	if IsQwenReasoningModel(name) {
		return name, true
	}
	if base, _, ok := ParseQwenReasoningEffortSuffix(name); ok {
		return base, true
	}
	if name == GPT6Astra || name == GPT6Astra+"-*" {
		return GPT6Astra, true
	}
	if base, suffix, known := splitGPT56Model(name); known {
		if suffix == "" || suffix == "*" {
			return base, true
		}
		base, _, _, ok := ParseGPT56ReasoningModelSuffix(name)
		return base, ok
	}
	base, _, _, ok := ParseGPT6AstraReasoningModelSuffix(name)
	return base, ok
}

// OpenAIReasoningModelNames expands base and wildcard registrations. Explicit
// variants stay explicit; a wildcard does not grant access to the bare base.
func OpenAIReasoningModelNames(name string) []string {
	base, known := OpenAIReasoningBaseModel(name)
	if !known || (name != base && name != base+"-*") {
		return []string{name}
	}
	if IsQwenReasoningModel(name) {
		names := make([]string, 0, 5)
		names = append(names, base)
		for _, effort := range []string{"none", "low", "medium", "xhigh"} {
			names = append(names, base+"-"+effort)
		}
		return names
	}
	names := make([]string, 0, 21)
	if name == base {
		names = append(names, base)
	}
	for _, mode := range []string{"", "standard", "pro"} {
		prefix := base
		if mode != "" {
			prefix += "-" + mode
			names = append(names, prefix)
		}
		for _, suffix := range OpenAIEffortSuffixes {
			candidate := prefix + suffix
			if _, _, _, ok := ParseOpenAIReasoningModelSuffix(candidate); ok {
				names = append(names, candidate)
			}
		}
	}
	return names
}
