package prototype

import (
	"fmt"

	catalogdata "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/data"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/loader"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

const (
	DaggerGameID        uint32 = 0x000F4240
	DeterminationGameID uint32 = 0x8000EA60
)

func Data() (schema.Manifest, []schema.Resource) {
	manifest, resources, err := Load()
	if err != nil {
		panic(fmt.Sprintf("load embedded prototype catalog: %v", err))
	}
	return manifest, resources
}

func Load() (schema.Manifest, []schema.Resource, error) {
	data, err := loader.LoadFS(catalogdata.Files())
	if err != nil {
		return schema.Manifest{}, nil, err
	}
	return data.Manifest, data.Resources(), nil
}
