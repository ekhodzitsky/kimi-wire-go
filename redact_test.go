package wire

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
