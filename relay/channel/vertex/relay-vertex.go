package vertex

import "github.com/QuantumNous/new-api/common"

// GetModelRegion resolves the channel's region setting. A JSON object maps
// model names to regions; the requested name is tried first, then the
// upstream model, which is the suffix-free base for effort variants such as
// gemini-4-hydrogen-xhigh, and finally "default".
func GetModelRegion(other string, localModelName string, upstreamModelName string) string {
	if !common.IsJsonObject(other) {
		return other
	}
	m, err := common.StrToMap(other)
	if err != nil {
		return other // return original if parsing fails
	}
	for _, key := range []string{localModelName, upstreamModelName, "default"} {
		if v, ok := m[key].(string); ok && key != "" {
			return v
		}
	}
	return "global"
}
