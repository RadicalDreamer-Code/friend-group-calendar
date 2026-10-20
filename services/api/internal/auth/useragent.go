package auth

import "strings"

// describeUserAgent turns a browser user agent into a short label for the
// device list, like "Safari on iPhone". It only needs to be recognisable,
// not exact; unknown agents return "".
func describeUserAgent(ua string) string {
	browser := firstMatch(ua, []match{
		// Order matters: most browsers also claim to be Chrome and Safari.
		{"Edg", "Edge"},
		{"OPR/", "Opera"},
		{"SamsungBrowser/", "Samsung Internet"},
		{"FxiOS/", "Firefox"},
		{"Firefox/", "Firefox"},
		{"CriOS/", "Chrome"},
		{"Chrome/", "Chrome"},
		{"Safari/", "Safari"},
	})
	os := firstMatch(ua, []match{
		{"iPhone", "iPhone"},
		{"iPad", "iPad"},
		{"Android", "Android"},
		{"CrOS", "ChromeOS"},
		{"Mac OS X", "macOS"},
		{"Macintosh", "macOS"},
		{"Windows", "Windows"},
		{"Linux", "Linux"},
	})
	switch {
	case browser != "" && os != "":
		return browser + " on " + os
	case browser != "":
		return browser
	default:
		return os
	}
}

type match struct{ needle, label string }

func firstMatch(s string, ms []match) string {
	for _, m := range ms {
		if strings.Contains(s, m.needle) {
			return m.label
		}
	}
	return ""
}
