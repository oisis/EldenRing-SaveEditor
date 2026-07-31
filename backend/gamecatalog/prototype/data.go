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
	all := data.Resources()
	resources := make([]schema.Resource, 0, 2)
	for _, gameID := range []uint32{DaggerGameID, DeterminationGameID} {
		found := false
		for _, resource := range all {
			if resource.Item == nil || resource.Item.GameID.Value != gameID {
				continue
			}
			resource.ID = schema.ResourceID(len(resources) + 1)
			resources = append(resources, resource)
			found = true
			break
		}
		if !found {
			return schema.Manifest{}, nil, fmt.Errorf(
				"prototype item 0x%08X is missing",
				gameID,
			)
		}
	}
	return data.Manifest, resources, nil
}
