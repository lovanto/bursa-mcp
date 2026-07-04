package fetcher

import "testing"

func TestLooksLikeChallenge(t *testing.T) {
	cases := []struct {
		name        string
		contentType string
		body        string
		want        bool
	}{
		{"html content-type", "text/html; charset=utf-8", `{"ok":true}`, true},
		{"doctype body", "application/octet-stream", "<!DOCTYPE html><html>...", true},
		{"just a moment", "", "<html><head><title>Just a moment...</title>", true},
		{"valid json", "application/json", `{"KodeEmiten":"BBCA","replies":[]}`, false},
		{"json no content-type", "", `{"a":1}`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := looksLikeChallenge(tc.contentType, []byte(tc.body)); got != tc.want {
				t.Errorf("looksLikeChallenge(%q, %q) = %v, want %v", tc.contentType, tc.body, got, tc.want)
			}
		})
	}
}

func TestConfigDefaults(t *testing.T) {
	c := Config{}.withDefaults()
	if c.MinInterval <= 0 || c.MaxRetries <= 0 || c.TimeoutSeconds <= 0 {
		t.Fatalf("defaults not applied: %+v", c)
	}
}
