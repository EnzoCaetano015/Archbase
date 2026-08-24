// Package schemas exposes the versioned Archbase JSON Schemas.
package schemas

import "embed"

// FS contains all public contract schemas.
//
//go:embed *.schema.json
var FS embed.FS
