package inventory

import (
	"encoding/binary"
	"os"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

func TestGetInventoryResolvesBareArmorAndFilledPhysickOnBothPlatforms(t *testing.T) {
	for _, platform := range []string{"pc", "ps4"} {
		t.Run(platform, func(t *testing.T) {
			path := writeGetInventoryFixture(t, platform, true, getInventoryAnchorAt)
			slotBase := technicalInventorySlotBase(platform)
			rewriteTechnicalFixture(t, path, slotBase+0x24,
				slotBase+getInventoryAnchorAt+getInventoryCommonAt+getInventoryRecordSize)

			engine := saveengine.New()
			session, err := engine.LoadSave(path, platform)
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			result, err := GetInventory(
				engine, inventoryCatalog(t), session.SaveSessionID, getInventorySlot, "", 0, 0)
			if err != nil {
				t.Fatalf("GetInventory: %v", err)
			}
			assertTechnicalContainerRecords(t, result.Records)
		})
	}
}

func TestGetStorageResolvesBareArmorAndFilledPhysickOnBothPlatforms(t *testing.T) {
	for _, platform := range []string{"pc", "ps4"} {
		t.Run(platform, func(t *testing.T) {
			path := writeGetStorageFixture(t, platform, true, getStorageAnchorAt)
			slotBase := technicalStorageSlotBase(platform)
			countAt := int64(getStorageAnchorAt + getStorageProjectileCountAt)
			sectionAt := countAt + 4 + getStorageProjectiles*getStorageProjectileStride + getStorageBlocksBefore
			rewriteTechnicalFixture(t, path, slotBase+0x24,
				slotBase+sectionAt+getStorageCommonAt+getStorageRecordSize)

			engine := saveengine.New()
			session, err := engine.LoadSave(path, platform)
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			result, err := GetStorage(
				engine, inventoryCatalog(t), session.SaveSessionID, getStorageSlot, "", 0, 0)
			if err != nil {
				t.Fatalf("GetStorage: %v", err)
			}
			assertTechnicalStorageRecords(t, result.Records)
		})
	}
}

func technicalInventorySlotBase(platform string) int64 {
	if platform == "ps4" {
		return int64(getInventoryPS4HeaderSize) + getInventorySlot*getInventoryPS4SlotSize
	}
	return int64(getInventoryHeaderSize) + 0x10 + getInventorySlot*getInventorySlotBlockSize
}

func technicalStorageSlotBase(platform string) int64 {
	if platform == "ps4" {
		return int64(getStoragePS4HeaderSize) + getStorageSlot*getStoragePS4SlotSize
	}
	return int64(getStorageHeaderSize) + 0x10 + getStorageSlot*getStorageSlotBlockSize
}

// rewriteTechnicalFixture changes only temporary synthetic evidence: the armor
// GaItem record becomes Bare Head and the direct goods handle becomes the
// filled save-side Physick alias. The source fixtures and real saves stay
// untouched.
func rewriteTechnicalFixture(t *testing.T, path string, armorGameIDAt, physickHandleAt int64) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	binary.LittleEndian.PutUint32(data[armorGameIDAt:], 0x10002710)
	binary.LittleEndian.PutUint32(data[physickHandleAt:], 0xB00000FA)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("rewrite fixture: %v", err)
	}
}

func assertTechnicalContainerRecords(t *testing.T, records []InventoryRecord) {
	t.Helper()

	foundBare, foundPhysick := false, false
	for _, record := range records {
		switch record.GameID {
		case 0x10002710:
			foundBare = record.Key == "10002710"
		case 0x400000FA:
			foundPhysick = record.Key == "400000FB"
		}
	}
	if !foundBare || !foundPhysick {
		t.Errorf("records = %+v, want Bare Head key 10002710 and Physick alias key 400000FB", records)
	}
}

func assertTechnicalStorageRecords(t *testing.T, records []StorageRecord) {
	t.Helper()

	foundBare, foundPhysick := false, false
	for _, record := range records {
		switch record.GameID {
		case 0x10002710:
			foundBare = record.Key == "10002710"
		case 0x400000FA:
			foundPhysick = record.Key == "400000FB"
		}
	}
	if !foundBare || !foundPhysick {
		t.Errorf("records = %+v, want Bare Head key 10002710 and Physick alias key 400000FB", records)
	}
}
