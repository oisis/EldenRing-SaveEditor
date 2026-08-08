package migration

import (
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func validateGeneratedCatalog(catalog GeneratedCatalog) error {
	sources, err := schema.ValidateManifest(catalog.Manifest)
	if err != nil {
		return fmt.Errorf("validate manifest: %w", err)
	}
	if len(catalog.Resources) != 2810 {
		return fmt.Errorf("resource count = %d, want 2810", len(catalog.Resources))
	}
	seenRefs := make(map[schema.ResourceRef]struct{}, len(catalog.Resources))
	variantCount := 0
	aliasCount := 0
	gestureSlotCount := 0
	for _, resource := range catalog.Resources {
		ref := resource.Ref()
		if _, duplicate := seenRefs[ref]; duplicate {
			return fmt.Errorf("duplicate resource kind %q key %q", ref.Kind, ref.Key)
		}
		seenRefs[ref] = struct{}{}
		if err := schema.ValidateResource(resource, sources); err != nil {
			return fmt.Errorf("validate %s: %w", resource.Key, err)
		}
		variantCount += len(resource.Item.Variants)
		aliasCount += len(resource.Item.Aliases)
		if resource.Item.Gesture != nil {
			gestureSlotCount += len(resource.Item.Gesture.Slots)
		}
	}
	if variantCount != 3624 {
		return fmt.Errorf("variant count = %d, want 3624", variantCount)
	}
	if aliasCount != 37 {
		return fmt.Errorf("alias count = %d, want 37", aliasCount)
	}
	if gestureSlotCount != 57 {
		return fmt.Errorf("gesture slot count = %d, want 57", gestureSlotCount)
	}
	if _, err := gamecatalog.New(catalog.Manifest, catalog.Resources); err != nil {
		return fmt.Errorf("build runtime catalog: %w", err)
	}
	return nil
}
