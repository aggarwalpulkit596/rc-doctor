// Package redact keeps API keys out of rc-doctor's output.
//
// rc-doctor prints a key's prefix and first four characters, never the whole
// key, in every format: terminal text, JSON, SARIF and fix previews. CI logs
// are often shared more widely than the code they build.
package redact

import "regexp"

// keyShaped matches tokens shaped like RevenueCat keys: a short lowercase
// prefix, an underscore and a long alphanumeric token.
var keyShaped = regexp.MustCompile(`\b([a-z]{2,4}_[A-Za-z0-9]{4})[A-Za-z0-9]{12,}`)

// Keys masks every key-shaped token in s: "appl_AbCdEfGh..." becomes "appl_AbCd…".
func Keys(s string) string {
	return keyShaped.ReplaceAllString(s, "${1}…")
}
