// Package registrydata contains the official offline registry shipped with arc.
package registrydata

import "embed"

// FS contains the registry index and its pattern files.
//
//go:embed index.yaml dotnet next rules
var FS embed.FS
