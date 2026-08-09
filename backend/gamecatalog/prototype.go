package gamecatalog

import (
	catalogdata "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/data"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/prototype"
)

func NewPrototype() (*Catalog, error) {
	manifest, resources, err := prototype.Load()
	if err != nil {
		return nil, err
	}
	networkPresets, err := LoadNetworkParams(catalogdata.Files())
	if err != nil {
		return nil, err
	}
	return NewWithNetworkParams(manifest, resources, networkPresets)
}
