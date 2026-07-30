package gamecatalog

import "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/prototype"

func NewPrototype() (*Catalog, error) {
	manifest, resources := prototype.Data()
	return New(manifest, resources)
}
