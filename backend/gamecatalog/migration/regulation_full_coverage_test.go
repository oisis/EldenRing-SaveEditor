package migration

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLegacyPrimaryRegulationCoverage(t *testing.T) {
	regulation := readLocalRegulationFixture(t)
	snapshot := collectLegacySnapshot()
	missing := make([]uint32, 0, 2)
	matched := 0

	for _, item := range snapshot.Items {
		identity, err := primaryRegulationForLegacyItem(item)
		if err != nil {
			t.Fatalf("item 0x%08X: %v", item.ID, err)
		}
		_, exists, err := regulation.LookupFamilyRow(
			identity.Family,
			RegulationTableRolePrimary,
			identity.RowID,
		)
		if err != nil {
			t.Fatalf("item 0x%08X primary lookup: %v", item.ID, err)
		}
		if !exists {
			missing = append(missing, item.ID)
			continue
		}
		matched++
	}

	if matched != 3834 {
		t.Fatalf("primary matches = %d, want 3834; missing=%#v", matched, missing)
	}
	wantMissing := []uint32{0x40002341, 0x4000234E}
	if len(missing) != len(wantMissing) {
		t.Fatalf("primary missing IDs = %#v, want %#v", missing, wantMissing)
	}
	for index := range wantMissing {
		if missing[index] != wantMissing[index] {
			t.Fatalf("primary missing IDs = %#v, want %#v", missing, wantMissing)
		}
	}
}

func TestLegacySpellRegulationRecordsShareExactRawRowID(t *testing.T) {
	regulation := readLocalRegulationFixture(t)
	spellCount := 0

	for _, item := range collectLegacySnapshot().Items {
		if item.Category != "sorceries" && item.Category != "incantations" {
			continue
		}
		spellCount++
		rawRowID := item.ID & 0x0FFFFFFF
		magic, magicExists, err := regulation.LookupFamilyRow(
			RegulationFamilySpell,
			RegulationTableRolePrimary,
			rawRowID,
		)
		if err != nil || !magicExists {
			t.Fatalf("spell 0x%08X Magic(%d) = %+v, %t, %v", item.ID, rawRowID, magic, magicExists, err)
		}
		goods, goodsExists, err := regulation.LookupFamilyRow(
			RegulationFamilyGoods,
			RegulationTableRolePrimary,
			rawRowID,
		)
		if err != nil || !goodsExists {
			t.Fatalf("spell 0x%08X EquipParamGoods(%d) = %+v, %t, %v", item.ID, rawRowID, goods, goodsExists, err)
		}
		if magic.RawRowID != goods.RawRowID {
			t.Fatalf("spell 0x%08X raw IDs differ: Magic=%d Goods=%d", item.ID, magic.RawRowID, goods.RawRowID)
		}
	}

	if spellCount != 213 {
		t.Fatalf("legacy spell count = %d, want 213", spellCount)
	}
}

func readLocalRegulationFixture(t *testing.T) *RegulationData {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current test file")
	}
	csvDirectory := filepath.Join(
		filepath.Dir(currentFile),
		"..", "..", "..",
		"tmp", "regulation-bin-dump", "csv",
	)
	if _, err := os.Stat(csvDirectory); err != nil {
		if os.IsNotExist(err) {
			t.Skip("local proprietary regulation CSV fixture is not available")
		}
		t.Fatalf("stat regulation fixture: %v", err)
	}
	regulation, err := ReadRegulationCSVDirectory(csvDirectory)
	if err != nil {
		t.Fatalf("ReadRegulationCSVDirectory: %v", err)
	}
	return regulation
}
