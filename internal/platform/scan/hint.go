package scan

import (
	"strings"
	"unicode"
)

var (
	iosWords     = map[string]bool{"IOS": true, "APPLE": true, "APPSTORE": true, "MACOS": true}
	androidWords = map[string]bool{"ANDROID": true, "GOOGLE": true, "PLAY": true, "PLAYSTORE": true, "AMAZON": true}
)

// PlatformHint returns the platform a line of code or config names, from its
// identifiers: "ios" for REVENUECAT_IOS_KEY or revenueCatIosKey, "android" for
// REVENUECAT_ANDROID_KEY or playStoreKey, "" when it names neither or both.
// Pass the line without the key value, whose random characters could spell a word.
func PlatformHint(line string) string {
	ios, android := false, false
	for _, w := range words(line) {
		ios = ios || iosWords[w]
		android = android || androidWords[w]
	}
	switch {
	case ios && !android:
		return "ios"
	case android && !ios:
		return "android"
	}
	return ""
}

// words splits on anything that isn't a letter or digit and on camelCase
// boundaries, and upper-cases the parts: "rcIosKey_v2" -> RC, IOS, KEY, V2.
func words(s string) []string {
	var out []string
	var cur []rune
	flush := func() {
		if len(cur) > 0 {
			out = append(out, strings.ToUpper(string(cur)))
			cur = cur[:0]
		}
	}
	rs := []rune(s)
	for i, r := range rs {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			flush()
			continue
		}
		// A new word starts at lower->Upper ("rcIos") and at the last capital
		// of a run followed by lower case ("RCKey" -> RC, Key).
		if unicode.IsUpper(r) && len(cur) > 0 {
			prev := rs[i-1]
			nextLower := i+1 < len(rs) && unicode.IsLower(rs[i+1])
			if unicode.IsLower(prev) || unicode.IsDigit(prev) || (unicode.IsUpper(prev) && nextLower) {
				flush()
			}
		}
		cur = append(cur, r)
	}
	flush()
	return out
}
