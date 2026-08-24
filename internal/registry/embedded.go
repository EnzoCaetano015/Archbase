package registry

import registrydata "github.com/EnzoCaetano015/Archbase/registry"

func NewEmbeddedSource() (Source, error) {
	return newCatalogSource("official-embedded", registrydata.FS)
}
