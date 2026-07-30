package gamecatalog

import "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/prototype"

func NewPrototype() (*Catalog, error) {
	manifest, resources, err := prototype.Load()
	if err != nil {
		return nil, err
	}
	return New(manifest, resources)
}
