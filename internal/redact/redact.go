package redact

import (
	"bytes"
	"encoding/json"
	"regexp"
)

var (
	secretKeyPattern = regexp.MustCompile(`(?i)(?:^|[^a-z0-9])(api[_-]?key|token|secret|password|auth|access[_-]?key|private[_-]?key|session[_-]?token|bearer|authorization)(?:$|[^a-z0-9])`)

	secretPatterns = []*regexp.Regexp{
		// key = value / key: value / "key": "value" style assignments.
		regexp.MustCompile(`(?i)((?:api[_-]?key|token|secret|password|auth|access[_-]?key|private[_-]?key|session[_-]?token|bearer|authorization)\s*[:=]\s*)["']?[^"'\s]{8,}["']?`),
		// Authorization: Bearer/Basic/Token/API-Key <value>
		regexp.MustCompile(`(?i)(authorization\s*[:=]\s*(?:bearer|basic|token|api-key)\s+)[^\s]+`),
		// AWS Access Key ID.
		regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`),
		// GitHub personal access tokens.
		regexp.MustCompile(`\b(ghp_|github_pat_|gho_|ghu_|ghs_|ghr_)[a-zA-Z0-9_\-]+\b`),
		// JWT (three base64url segments, including padding).
		regexp.MustCompile(`\beyJ[a-zA-Z0-9_-]+\.[a-zA-Z0-9_-]+\.[a-zA-Z0-9_-]+=*`),
		// URL with embedded credentials.
		regexp.MustCompile(`(?i)(\bhttps?://[^:]+:)[^@]+(@[^\s]+)`),
		// PEM private keys.
		regexp.MustCompile(`(?s)-----BEGIN (?:RSA |EC |OPENSSH |DSA |ENCRYPTED )?PRIVATE KEY-----.*?-----END (?:RSA |EC |OPENSSH |DSA |ENCRYPTED )?PRIVATE KEY-----`),
		// PGP private key block.
		regexp.MustCompile(`(?s)-----BEGIN PGP PRIVATE KEY BLOCK-----.*?-----END PGP PRIVATE KEY BLOCK-----`),
	}

	secretReplacements = []string{
		"${1}***",
		"${1}***",
		"AKIA...REDACTED",
		"${1}***",
		"eyJ...REDACTED",
		"${1}***${2}",
		"[PEM_PRIVATE_KEY_REDACTED]",
		"[PGP_PRIVATE_KEY_BLOCK_REDACTED]",
	}
)

// RedactSecrets recursively scrubs secrets from a JSON-like value.
func RedactSecrets(v any) any {
	switch val := v.(type) {
	case string:
		return RedactString(val)
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
	case json.RawMessage:
		dec := json.NewDecoder(bytes.NewReader(val))
		dec.UseNumber()
		var inner any
		if err := dec.Decode(&inner); err != nil {
			return json.RawMessage(RedactString(string(val)))
		}
		redacted := RedactSecrets(inner)
		out, err := json.Marshal(redacted)
		if err != nil {
			return json.RawMessage(RedactString(string(val)))
		}
		return json.RawMessage(out)
	default:
		return v
	}
}

func RedactString(s string) string {
	result := s
	for i, re := range secretPatterns {
		result = re.ReplaceAllString(result, secretReplacements[i])
	}
	return result
}
