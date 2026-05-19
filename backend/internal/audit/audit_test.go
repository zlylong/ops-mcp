package audit

import "testing"

func TestMaskRedactsSensitiveKeys(t *testing.T) {
	t.Parallel()
	masked := Mask(map[string]any{
		"query":    "up",
		"password": "plain",
		"nested":   map[string]any{"apiToken": "secret", "safe": "ok"},
	})
	if masked["query"] != "up" {
		t.Fatalf("safe field masked unexpectedly")
	}
	if masked["password"] != "***MASKED***" {
		t.Fatalf("password was not masked: %#v", masked["password"])
	}
	nested := masked["nested"].(map[string]any)
	if nested["apiToken"] != "***MASKED***" {
		t.Fatalf("nested token was not masked: %#v", nested["apiToken"])
	}
	if nested["safe"] != "ok" {
		t.Fatalf("nested safe value changed")
	}
}
