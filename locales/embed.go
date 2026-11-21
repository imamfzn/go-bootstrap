// Package locales embeds the translation catalogs into the binary.
// Each file is "<locale>.json" and maps an error code to its message.
package locales

import "embed"

// FS holds the catalogs at its root.
//
//go:embed *.json
var FS embed.FS
