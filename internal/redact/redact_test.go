package redact

import (
	"testing"
)

func TestRedactSecretsMap(t *testing.T) {
	input := map[string]any{
		"url":     "https://example.com",
		"api_key": "super-secret",
	}
	result := RedactSecrets(input).(map[string]any)
	if result["url"] != "https://example.com" {
		t.Fatal("url should not be redacted")
	}
	if result["api_key"] == "super-secret" {
		t.Fatal("api_key should be redacted")
	}
}

func TestRedactStringPatterns(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"Authorization: Bearer sk-abc123xyz", "Authorization: Bearer ***"},
		{"token=supersecrettokenvalue", "token=***"},
		{"AKIAIOSFODNN7EXAMPLE", "AKIA...REDACTED"},
		{"ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx", "ghp_***"},
		{"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U", "eyJ...REDACTED"},
		{"no secret here", "no secret here"},
	}
	for _, tc := range cases {
		got := RedactString(tc.in)
		if got != tc.want {
			t.Errorf("RedactString(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
