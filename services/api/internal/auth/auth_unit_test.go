package auth

import "testing"

func TestNormalizeEmail(t *testing.T) {
	valid := map[string]string{
		"  Ada@Example.COM ":  "ada@example.com",
		"first.last+tag@a.io": "first.last+tag@a.io",
	}
	for in, want := range valid {
		got, err := NormalizeEmail(in)
		if err != nil || got != want {
			t.Errorf("NormalizeEmail(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, in := range []string{"", "ada", "ada@localhost", "ada@example.", "Ada <ada@example.com>", "a@b.c, d@e.f"} {
		if _, err := NormalizeEmail(in); err == nil {
			t.Errorf("NormalizeEmail(%q) should fail", in)
		}
	}
}

func TestDescribeUserAgent(t *testing.T) {
	cases := map[string]string{
		"Mozilla/5.0 (iPhone; CPU iPhone OS 26_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/26.0 Mobile/15E148 Safari/604.1": "Safari on iPhone",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36":                   "Chrome on macOS",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36 Edg/141.0.0.0":           "Edge on Windows",
		"Mozilla/5.0 (X11; Linux x86_64; rv:144.0) Gecko/20100101 Firefox/144.0":                                                                  "Firefox on Linux",
		"Mozilla/5.0 (Linux; Android 16; Pixel 9) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Mobile Safari/537.36":                   "Chrome on Android",
		"okhttp/4.12.0": "",
	}
	for ua, want := range cases {
		if got := describeUserAgent(ua); got != want {
			t.Errorf("describeUserAgent(%q) = %q, want %q", ua, got, want)
		}
	}
}

func TestTruncateKeepsRunesWhole(t *testing.T) {
	if got := truncate("ab€", 3); got != "ab" {
		t.Fatalf("truncate = %q, want %q", got, "ab")
	}
}
