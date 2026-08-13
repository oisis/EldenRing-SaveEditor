package equipment

import (
	"reflect"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestSetEquippedSpellsResolvesSpellsAndUpdatesRevision(t *testing.T) {
	engine, sessionID := loadEquippedSpellsSession(t, []uint32{rawGlintstonePebble})
	catalog := newEquippedSpellsCatalog(t)

	pebbleRef := &schema.ResourceRef{Kind: schema.ResourceKindItem, Key: "40000FA0"}
	cometRef := &schema.ResourceRef{Kind: schema.ResourceKindItem, Key: "40001068"}

	result, err := SetEquippedSpells(
		engine,
		catalog,
		sessionID,
		getEquippedSpellsSlot,
		[]*schema.ResourceRef{pebbleRef, cometRef},
		"0",
	)
	if err != nil {
		t.Fatalf("SetEquippedSpells: %v", err)
	}
	if result.SaveSessionID != sessionID || result.SaveRevision != "1" || result.CharacterID != getEquippedSpellsSlot {
		t.Fatalf("result header = %+v, want revision 1", result)
	}
	wantRefs := []*schema.ResourceRef{
		{Kind: schema.ResourceKindItem, Key: "40000FA0"},
		{Kind: schema.ResourceKindItem, Key: "40001068"},
	}
	if !reflect.DeepEqual(result.OrderedResources, wantRefs) {
		t.Errorf("orderedResources = %+v, want %+v", result.OrderedResources, wantRefs)
	}
	if result.UsedMemorySlots != 4 || result.AvailableMemorySlots != 5 {
		t.Errorf("used/available = %d/%d, want 4/5", result.UsedMemorySlots, result.AvailableMemorySlots)
	}

	// Verify getter returns updated 12 slots with the 2 spells at index 0 and 1
	got, err := GetEquippedSpells(engine, catalog, sessionID, getEquippedSpellsSlot)
	if err != nil {
		t.Fatalf("GetEquippedSpells: %v", err)
	}
	if len(got.Spells) != 12 {
		t.Fatalf("getter returned %d spells, want 12", len(got.Spells))
	}
	if got.Spells[0].ResourceKey != "40000FA0" || got.Spells[1].ResourceKey != "40001068" {
		t.Errorf("getter spells = %+v, want pebble and comet", got.Spells[:2])
	}
	if got.UsedMemorySlots != 4 {
		t.Errorf("getter usedMemorySlots = %d, want 4", got.UsedMemorySlots)
	}
}

func TestSetEquippedSpellsClearsLoadout(t *testing.T) {
	engine, sessionID := loadEquippedSpellsSession(t, []uint32{rawGlintstonePebble})
	catalog := newEquippedSpellsCatalog(t)

	result, err := SetEquippedSpells(
		engine,
		catalog,
		sessionID,
		getEquippedSpellsSlot,
		[]*schema.ResourceRef{},
		"0",
	)
	if err != nil {
		t.Fatalf("SetEquippedSpells empty: %v", err)
	}
	if result.SaveRevision != "1" || result.UsedMemorySlots != 0 {
		t.Fatalf("result = %+v, want revision 1 and used 0", result)
	}
	if len(result.OrderedResources) != 0 || result.OrderedResources == nil {
		t.Fatalf("orderedResources = %v, want non-nil empty slice", result.OrderedResources)
	}

	got, err := GetEquippedSpells(engine, catalog, sessionID, getEquippedSpellsSlot)
	if err != nil {
		t.Fatalf("GetEquippedSpells after clear: %v", err)
	}
	for i, slot := range got.Spells {
		if slot.RawMagicParamID != 0xFFFFFFFF {
			t.Errorf("slot %d raw ID = 0x%08X, want 0xFFFFFFFF", i, slot.RawMagicParamID)
		}
	}
}

func TestSetEquippedSpellsRejectsInvalidRequests(t *testing.T) {
	engine, sessionID := loadEquippedSpellsSession(t, []uint32{rawGlintstonePebble})
	catalog := newEquippedSpellsCatalog(t)

	pebbleRef := &schema.ResourceRef{Kind: schema.ResourceKindItem, Key: "40000FA0"}
	nonSpellGoodsRef := &schema.ResourceRef{Kind: schema.ResourceKindItem, Key: "4000272E"}
	unknownRef := &schema.ResourceRef{Kind: schema.ResourceKindItem, Key: "40000FFF"}

	tests := []struct {
		name      string
		resources []*schema.ResourceRef
		rev       string
		want      string
	}{
		{
			name:      "nil element in orderedResources",
			resources: []*schema.ResourceRef{pebbleRef, nil},
			rev:       "0",
			want:      "orderedResources[1] cannot be nil",
		},
		{
			name:      "non-spell resource",
			resources: []*schema.ResourceRef{nonSpellGoodsRef},
			rev:       "0",
			want:      `orderedResources[0]: resource kind "item" key "4000272E" has item family "goods", want "spell"`,
		},
		{
			name:      "unknown spell resource",
			resources: []*schema.ResourceRef{unknownRef},
			rev:       "0",
			want:      `orderedResources[0]: unknown resource key "40000FFF" in kind "item"`,
		},
		{
			name:      "duplicate spell",
			resources: []*schema.ResourceRef{pebbleRef, pebbleRef},
			rev:       "0",
			want:      "orderedResources[1]: spell 0x40000FA0 is duplicated",
		},
		{
			name:      "stale revision",
			resources: []*schema.ResourceRef{pebbleRef},
			rev:       "99",
			want:      `expectedRevision "99" does not match the current saveRevision "0"`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := SetEquippedSpells(engine, catalog, sessionID, getEquippedSpellsSlot, tc.resources, tc.rev)
			if err == nil {
				t.Fatalf("SetEquippedSpells accepted invalid input")
			}
			if err.Error() != tc.want {
				t.Errorf("error = %q, want %q", err.Error(), tc.want)
			}
		})
	}
}

func TestSetEquippedSpellsRejectsMissingDependencies(t *testing.T) {
	engine, sessionID := loadEquippedSpellsSession(t, []uint32{rawGlintstonePebble})
	catalog := newEquippedSpellsCatalog(t)

	t.Run("nil engine", func(t *testing.T) {
		_, err := SetEquippedSpells(nil, catalog, sessionID, getEquippedSpellsSlot, nil, "0")
		if err == nil || err.Error() != "save engine is not available" {
			t.Errorf("error = %v, want 'save engine is not available'", err)
		}
	})

	t.Run("nil catalog", func(t *testing.T) {
		_, err := SetEquippedSpells(engine, nil, sessionID, getEquippedSpellsSlot, nil, "0")
		if err == nil || err.Error() != "game catalog is not available" {
			t.Errorf("error = %v, want 'game catalog is not available'", err)
		}
	})
}
