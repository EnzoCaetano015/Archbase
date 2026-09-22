// Package registrydata contains the official offline registry shipped with arc.
package registrydata

import "embed"

// FS contains the registry index and its pattern files.
//
//go:embed index.yaml astro dotnet nestjs next next-feature python react react-tailwind rules
var FS embed.FS
