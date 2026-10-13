package redact

import "testing"

func TestKeys(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"appl_AbCdEfGhIjKlMnOpQrSt", "appl_AbCd…"},
		{`Purchases.configure(withAPIKey: "sk_AbCdEfGhIjKlMnOpQrSt")`, `Purchases.configure(withAPIKey: "sk_AbCd…")`},
		{"two keys: goog_ZyXwVuTsRqPoNmLkJiHg and amzn_1234567890abcdefgh", "two keys: goog_ZyXw… and amzn_1234…"},
		{"short goog_abc stays", "short goog_abc stays"},
		{"no key here", "no key here"},
	} {
		if got := Keys(tc.in); got != tc.want {
			t.Errorf("Keys(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
