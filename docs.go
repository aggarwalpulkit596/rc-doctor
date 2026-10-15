// Package rcdoctor holds files that ship inside the rc-doctor binary.
package rcdoctor

import "embed"

// RuleDocs are the rule documentation pages, docs/rules/<rule-id>.md, shown by
// `rc-doctor explain <rule-id>` and linked from every report.
//
//go:embed docs/rules/*.md
var RuleDocs embed.FS
