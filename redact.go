package wire

import (
	"regexp"
	"strings"
)

var (
	secretKeyPattern = regexp.MustCompile(`(?i)(api[_-]?key|token|secret|password|auth)`)
	secretPatterns   = []*regexp.Regexp{
		regexp.MustCompile(`(?i)["']?(api[_-]?key|token|secret|password|auth)["']?\s*[:=]\s*["']?([a-zA-Z0-9_\-]{8,})["']?`),
	}
)

// RedactSecrets recursively scrubs secrets from a JSON-like value.
func RedactSecrets(v any) any {
	switch val := v.(type) {
	case string:
		return redactString(val)
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, v2 := range val {
			if secretKeyPattern.MatchString(k) {
				out[k] = "***"
			} else {
				out[k] = RedactSecrets(v2)
			}
		}
		return out
	case []any:
		out := make([]any, len(val))
		for i, v2 := range val {
			out[i] = RedactSecrets(v2)
		}
		return out
	default:
		return v
	}
}

func redactString(s string) string {
	result := s
	for _, re := range secretPatterns {
		result = re.ReplaceAllString(result, "${1}=***")
	}
	if strings.Contains(strings.ToLower(s), "authorization") {
		parts := strings.SplitN(s, ":", 2)
		if len(parts) == 2 {
			return parts[0] + ": ***"
		}
		parts = strings.SplitN(s, " ", 2)
		if len(parts) == 2 {
			return parts[0] + " ***"
		}
	}
	return result
}
