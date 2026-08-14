package saveengine

import (
	"reflect"
	"testing"
)

func TestGetInventoryGoodsPresenceReadsBothRepresentationsOnBothPlatforms(t *testing.T) {
	const (
		directGameID  = uint32(0x4000218E)
		encodedGameID = uint32(0x4000230A)
		missingGameID = uint32(0x4000230B)
	)
	for _, platform := range []Platform{PlatformPC, PlatformPS4} {
		t.Run(string(platform), func(t *testing.T) {
			content := inventoryTestActiveFixture(platform, 3, 0x1A7)
			content.common = []inventoryTestRow{{
				index: 2, handle: directGameID, rawQuantity: 1,
			}}
			content.key = []inventoryTestRow{
				{index: 4, handle: 0xB000230A, rawQuantity: 1},
				{index: 5, handle: 0xB000230B, rawQuantity: 0},
			}
			engine := New()
			loaded, err := engine.LoadSave(writeInventoryFixture(t, content), string(platform))
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}

			got, err := engine.GetInventoryGoodsPresence(
				loaded.SaveSessionID, content.slot,
				[]uint32{directGameID, encodedGameID, missingGameID})
			if err != nil {
				t.Fatalf("GetInventoryGoodsPresence: %v", err)
			}
			want := map[uint32]bool{
				directGameID: true, encodedGameID: true, missingGameID: false,
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("presence = %#v, want %#v", got, want)
			}
		})
	}
}

func TestGetInventoryGoodsPresenceDoesNotReadResidualInventory(t *testing.T) {
	content := inventoryTestFixture{platform: PlatformPC, slot: 2, flag: 0, noAnchor: true}
	engine := New()
	loaded, err := engine.LoadSave(writeInventoryFixture(t, content), "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	got, err := engine.GetInventoryGoodsPresence(
		loaded.SaveSessionID, content.slot, []uint32{0x4000218E})
	if err != nil {
		t.Fatalf("GetInventoryGoodsPresence: %v", err)
	}
	if !reflect.DeepEqual(got, map[uint32]bool{0x4000218E: false}) {
		t.Errorf("presence = %#v", got)
	}
}
