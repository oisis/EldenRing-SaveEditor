package world

import (
	"os"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// The three colosseum unlock flags of the stored catalog. The fixture sets the
// first and the last one and leaves the middle one clear, so a shifted byte or
// an inverted bit direction cannot pass unnoticed.
const (
	getColosseumsCaelidFlag   = 60350
	getColosseumsLimgraveFlag = 60360
	getColosseumsRoyalFlag    = 60370
)

// writeGetColosseumsFixture reuses the synthetic PC container of the cookbook
// tests and sets the two block-60 flags directly, because the cookbook fixture
// only places the blocks its own flags live in.
func writeGetColosseumsFixture(t *testing.T, active bool) string {
	t.Helper()

	path := writeGetCookbooksFixture(t, nil, active)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	anchorBase := int64(getCookbooksHeaderSize) + 0x10 +
		getCookbooksSlot*getCookbooksSlotBlockSize + getCookbooksAnchorAt
	for _, id := range []uint32{getColosseumsCaelidFlag, getColosseumsRoyalFlag} {
		index := int64(id % 1000)
		offset := 10*getCookbooksBlockSize + index/8
		data[anchorBase+getCookbooksSectionAt+offset] |= 1 << uint8(7-index%8)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func loadColosseumsSession(t *testing.T, active bool) (*saveengine.Engine, string) {
	t.Helper()

	engine := saveengine.New()
	loaded, err := engine.LoadSave(writeGetColosseumsFixture(t, active), "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	return engine, loaded.SaveSessionID
}

// patchColosseumDocument copies the colosseum document of one stored resource
// and applies change to the copy, so the shared stored documents stay untouched.
func patchColosseumDocument(
	t *testing.T, resources []schema.Resource, key string, change func(*schema.ColosseumDocument),
) []schema.Resource {
	t.Helper()

	patched := 0
	for index := range resources {
		if resources[index].Key != key || resources[index].Colosseum == nil {
			continue
		}
		document := *resources[index].Colosseum
		change(&document)
		resources[index].Colosseum = &document
		patched++
	}
	if patched != 1 {
		t.Fatalf("patched %d colosseums of %q, want 1", patched, key)
	}
	return resources
}

func TestGetColosseumsDeclaresExactlyThreeCatalogResources(t *testing.T) {
	gameCatalog := newCookbooksCatalog(t)

	want := map[string]uint32{
		"caelid_colosseum":   getColosseumsCaelidFlag,
		"limgrave_colosseum": getColosseumsLimgraveFlag,
		"royal_colosseum":    getColosseumsRoyalFlag,
	}
	wantNames := map[string]string{
		"caelid_colosseum":   "Caelid Colosseum",
		"limgrave_colosseum": "Limgrave Colosseum",
		"royal_colosseum":    "Royal Colosseum",
	}
	found := 0
	for _, summary := range gameCatalog.ResourceSummaries() {
		if summary.Kind != schema.ResourceKindColosseum {
			continue
		}
		found++
		resource, err := gameCatalog.ResourceByKindAndKey(summary.Kind, summary.Key)
		if err != nil {
			t.Fatalf("ResourceByKindAndKey(%q): %v", summary.Key, err)
		}
		if resource.Item != nil {
			t.Errorf("colosseum %q carries an item document", summary.Key)
		}
		if resource.Colosseum == nil {
			t.Fatalf("colosseum %q carries no colosseum document", summary.Key)
		}
		if got := resource.Colosseum.Name; !got.Known || got.Value != wantNames[summary.Key] {
			t.Errorf("colosseum %q name = %q (known %t), want %q",
				summary.Key, got.Value, got.Known, wantNames[summary.Key])
		}
		if got := resource.Colosseum.UnlockEventFlagID; !got.Known || got.Value != want[summary.Key] {
			t.Errorf("colosseum %q event flag = %d (known %t), want %d",
				summary.Key, got.Value, got.Known, want[summary.Key])
		}
	}
	if found != len(want) {
		t.Fatalf("catalog declares %d colosseums, want %d", found, len(want))
	}
}

func TestGetColosseumsReportsFlagState(t *testing.T) {
	engine, sessionID := loadColosseumsSession(t, true)

	result, err := GetColosseums(engine, newCookbooksCatalog(t), sessionID, getCookbooksSlot)
	if err != nil {
		t.Fatalf("GetColosseums: %v", err)
	}
	if !result.Active || result.SaveSessionID != sessionID || result.CharacterID != getCookbooksSlot {
		t.Fatalf("result = %+v, want the active requested slot", result)
	}
	want := []ColosseumEntry{
		{Kind: schema.ResourceKindColosseum, Key: "caelid_colosseum", Name: "Caelid Colosseum", Unlocked: true},
		{Kind: schema.ResourceKindColosseum, Key: "limgrave_colosseum", Name: "Limgrave Colosseum", Unlocked: false},
		{Kind: schema.ResourceKindColosseum, Key: "royal_colosseum", Name: "Royal Colosseum", Unlocked: true},
	}
	if len(result.Colosseums) != len(want) {
		t.Fatalf("result carries %d colosseums, want %d", len(result.Colosseums), len(want))
	}
	for index, entry := range result.Colosseums {
		if entry != want[index] {
			t.Errorf("entry %d = %+v, want %+v", index, entry, want[index])
		}
	}
}

func TestGetColosseumsDoesNotReadResidualState(t *testing.T) {
	engine, sessionID := loadColosseumsSession(t, false)

	result, err := GetColosseums(engine, newCookbooksCatalog(t), sessionID, getCookbooksSlot)
	if err != nil {
		t.Fatalf("GetColosseums: %v", err)
	}
	if result.Active || len(result.Colosseums) != 3 {
		t.Fatalf("result active/count = %t/%d, want false/3", result.Active, len(result.Colosseums))
	}
	for _, entry := range result.Colosseums {
		if entry.Unlocked {
			t.Errorf("residual slot reports unlocked entry %+v", entry)
		}
	}
}

func TestGetColosseumsRejectsInvalidInput(t *testing.T) {
	engine, sessionID := loadColosseumsSession(t, true)
	gameCatalog := newCookbooksCatalog(t)

	if _, err := GetColosseums(nil, gameCatalog, sessionID, getCookbooksSlot); err == nil {
		t.Error("missing SaveEngine was accepted")
	}
	if _, err := GetColosseums(engine, nil, sessionID, getCookbooksSlot); err == nil {
		t.Error("missing GameCatalog was accepted")
	}
	if _, err := GetColosseums(engine, gameCatalog, "", getCookbooksSlot); err == nil {
		t.Error("empty saveSessionID was accepted")
	}
	if _, err := GetColosseums(engine, gameCatalog, sessionID+"x", getCookbooksSlot); err == nil {
		t.Error("unknown saveSessionID was accepted")
	}
	for _, characterID := range []int{-1, 10} {
		if _, err := GetColosseums(engine, gameCatalog, sessionID, characterID); err == nil {
			t.Errorf("characterID %d was accepted", characterID)
		}
	}
}

func TestGetColosseumsRejectsDuplicateEventFlag(t *testing.T) {
	engine, sessionID := loadColosseumsSession(t, true)
	resources := patchColosseumDocument(
		t, storedCookbookResources(t), "royal_colosseum",
		func(document *schema.ColosseumDocument) {
			document.UnlockEventFlagID.Value = getColosseumsCaelidFlag
		},
	)

	_, err := GetColosseums(
		engine, cookbooksCatalogOf(t, resources), sessionID, getCookbooksSlot)
	if err == nil || !strings.Contains(err.Error(), "both declare event flag") {
		t.Fatalf("duplicate event flag error = %v", err)
	}
}

// A flag outside a supported block must surface the SaveEngine rejection, which
// proves the endpoint asks SaveEngine to place every flag instead of decoding
// the bitfield itself.
func TestGetColosseumsDelegatesFlagDecodingToSaveEngine(t *testing.T) {
	engine, sessionID := loadColosseumsSession(t, true)
	resources := patchColosseumDocument(
		t, storedCookbookResources(t), "royal_colosseum",
		func(document *schema.ColosseumDocument) {
			document.UnlockEventFlagID.Value = 99000
		},
	)

	_, err := GetColosseums(
		engine, cookbooksCatalogOf(t, resources), sessionID, getCookbooksSlot)
	if err == nil || !strings.Contains(err.Error(), "which this reader does not support") {
		t.Fatalf("unsupported block error = %v", err)
	}
}

func TestGetColosseumsSortsByNameThenKey(t *testing.T) {
	engine, sessionID := loadColosseumsSession(t, true)
	resources := storedCookbookResources(t)
	for _, key := range []string{"caelid_colosseum", "limgrave_colosseum", "royal_colosseum"} {
		resources = patchColosseumDocument(t, resources, key,
			func(document *schema.ColosseumDocument) { document.Name.Value = "Colosseum" })
	}

	result, err := GetColosseums(
		engine, cookbooksCatalogOf(t, resources), sessionID, getCookbooksSlot)
	if err != nil {
		t.Fatalf("GetColosseums: %v", err)
	}
	want := []string{"caelid_colosseum", "limgrave_colosseum", "royal_colosseum"}
	for index, entry := range result.Colosseums {
		if entry.Key != want[index] {
			t.Fatalf("entry %d key = %q, want %q", index, entry.Key, want[index])
		}
	}
}

// A catalog whose colosseum facts are incomplete or whose union is contradictory
// must never reach the getter: the catalog itself rejects it.
func TestColosseumCatalogFailsClosed(t *testing.T) {
	manifest := storedCookbookCatalogData(t).Manifest
	for name, change := range map[string]func(*schema.Resource){
		"missing name": func(resource *schema.Resource) {
			resource.Colosseum.Name = schema.Fact[string]{Provenance: resource.Colosseum.Name.Provenance}
		},
		"missing event flag": func(resource *schema.Resource) {
			resource.Colosseum.UnlockEventFlagID = schema.Fact[uint32]{
				Provenance: resource.Colosseum.UnlockEventFlagID.Provenance,
			}
		},
		"missing document": func(resource *schema.Resource) { resource.Colosseum = nil },
		"conflicting union": func(resource *schema.Resource) {
			resource.Item = &schema.ItemDocument{}
		},
	} {
		resources := storedCookbookResources(t)
		patched := 0
		for index := range resources {
			if resources[index].Key != "royal_colosseum" {
				continue
			}
			document := *resources[index].Colosseum
			resources[index].Colosseum = &document
			change(&resources[index])
			patched++
		}
		if patched != 1 {
			t.Fatalf("%s: patched %d resources, want 1", name, patched)
		}
		if _, err := gamecatalog.New(manifest, resources); err == nil {
			t.Errorf("%s: catalog accepted an invalid colosseum resource", name)
		}
	}
}
