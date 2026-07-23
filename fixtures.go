// Package fixtures exposes the embedded provider response fixtures from testdata/.
package fixtures

import "embed"

// FS holds the embedded provider JSON response fixtures.
//
//go:embed testdata/*.json
var FS embed.FS
