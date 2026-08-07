package gamecatalog_test

import (
	"encoding/json"
	"io/fs"
	"strings"
	"testing"

	catalogdata "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/data"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

// removedStorageGameLimitFields are the duplicated technical storage limits that
// item.storage no longer carries. The effective limits in maxInventory and
// maxStorage are the single source of truth.
var removedStorageGameLimitFields = []string{
	"gameMaxInventory",
	"gameMaxStorage",
	"gameMaxInventory-sfv",
	"gameMaxStorage-sfv",
}

// TestEmbeddedStorageHasNoGameLimitFields proves no shipped document or variant
// storage block retains the removed technical limits, while the identically
// named RelatedTechnicalRecord fields survive untouched.
func TestEmbeddedStorageHasNoGameLimitFields(t *testing.T) {
	technicalRecords := 0
	err := fs.WalkDir(catalogdata.Files(), "items", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}
		raw, err := fs.ReadFile(catalogdata.Files(), path)
		if err != nil {
			return err
		}
		var document struct {
			Item struct {
				Storage  map[string]json.RawMessage `json:"storage"`
				Variants []struct {
					Data struct {
						Storage map[string]json.RawMessage `json:"storage"`
					} `json:"data"`
				} `json:"variants"`
				RelatedTechnicalRecords []map[string]json.RawMessage `json:"relatedTechnicalRecords"`
			} `json:"item"`
		}
		if err := json.Unmarshal(raw, &document); err != nil {
			return err
		}
		assertNoStorageGameLimits(t, path, document.Item.Storage)
		for _, variant := range document.Item.Variants {
			assertNoStorageGameLimits(t, path, variant.Data.Storage)
		}
		for _, record := range document.Item.RelatedTechnicalRecords {
			if _, exists := record["gameMaxInventory"]; !exists {
				t.Errorf("%s technical record lost gameMaxInventory", path)
			}
			if _, exists := record["gameMaxStorage"]; !exists {
				t.Errorf("%s technical record lost gameMaxStorage", path)
			}
			technicalRecords++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk embedded documents: %v", err)
	}
	if technicalRecords == 0 {
		t.Fatal("no related technical record was scanned; the scan proves nothing")
	}
}

func assertNoStorageGameLimits(
	t *testing.T,
	path string,
	storage map[string]json.RawMessage,
) {
	t.Helper()
	if storage == nil {
		return
	}
	for _, field := range removedStorageGameLimitFields {
		if _, exists := storage[field]; exists {
			t.Errorf("%s storage still contains %q", path, field)
		}
	}
}

// TestTheRingGestureStorageLimitsComeFromRegulation pins the slot-only gesture
// The Ring to the EquipParamGoods row 9049 limits, and keeps the neighbouring
// slot-only gesture without a Goods row unknown.
func TestTheRingGestureStorageLimitsComeFromRegulation(t *testing.T) {
	const theRingGameID = 0x40002359
	const slotOnlyGestureWithoutGoodsRow = 0x40002354
	const goodsSource = schema.SourceID("regulation_equip_param_goods")

	storages := indexEmbeddedStorage(t)

	theRing, exists := storages[theRingGameID]
	if !exists {
		t.Fatalf("gesture 0x%08X is missing from the embedded catalog", theRingGameID)
	}
	if !theRing.MaxInventory.Known || theRing.MaxInventory.Value != 1 {
		t.Errorf("The Ring maxInventory = %+v, want known 1", theRing.MaxInventory)
	}
	if !theRing.MaxStorage.Known || theRing.MaxStorage.Value != 1 {
		t.Errorf("The Ring maxStorage = %+v, want known 1", theRing.MaxStorage)
	}
	if theRing.MaxInventory.Provenance.Source != goodsSource ||
		theRing.MaxStorage.Provenance.Source != goodsSource {
		t.Errorf(
			"The Ring storage sources = %q/%q, want %q",
			theRing.MaxInventory.Provenance.Source,
			theRing.MaxStorage.Provenance.Source,
			goodsSource,
		)
	}

	withoutGoods, exists := storages[slotOnlyGestureWithoutGoodsRow]
	if !exists {
		t.Fatalf("gesture 0x%08X is missing from the embedded catalog", slotOnlyGestureWithoutGoodsRow)
	}
	if withoutGoods.MaxInventory.Known || withoutGoods.MaxStorage.Known {
		t.Errorf(
			"gesture 0x%08X fabricated storage limits = %#v",
			slotOnlyGestureWithoutGoodsRow,
			withoutGoods,
		)
	}
}

func indexEmbeddedStorage(t *testing.T) map[uint32]schema.ItemStorage {
	t.Helper()
	resources := loadEmbeddedResources(t)
	storages := make(map[uint32]schema.ItemStorage, len(resources))
	for _, resource := range resources {
		storages[resource.Item.GameID.Value] = resource.Item.Storage
		for _, variant := range resource.Item.Variants {
			storages[variant.GameID.Value] = variant.Data.Storage
		}
	}
	return storages
}
