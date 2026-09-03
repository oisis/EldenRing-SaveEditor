package equipment

import (
	"encoding/binary"
	"os"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// The shared Pouch endpoint fixture already owns exactly the three records this
// getter has to tell apart: one goods item the catalog allows in a Quick Item
// and a Pouch slot, one goods item with no confirmed equipment capability, and
// one talisman. Reusing it keeps the compatibility rule under test instead of a
// second hand-built container.
const (
	candidatesThrowingDaggerKey = "400006A4"
	candidatesMoonKey           = "20000474"

	// A Crystal Tear is a key item, so the one owned Physick candidate of this
	// fixture is stored in Inventory key and not in Inventory common. It proves
	// the getter reads the same two sections SetPhysickMixture accepts.
	candidatesStonebarbTearKey    = "40002B12"
	candidatesStonebarbTearHandle = uint32(0xB0002B12)

	// The physical layout of the key section behind the common one, restated
	// here so the fixture writes a key record without borrowing a private
	// SaveEngine constant.
	candidatesKeySectionAt = 0xA80*12 + 4
)

func loadEquipmentCandidatesFixture(t *testing.T) (*saveengine.Engine, string, *gamecatalog.Catalog) {
	t.Helper()
	path, platform := writeSetPouchEndpointFixture(t)

	// The shared Pouch fixture owns only common records. One Crystal Tear is
	// added to the key section here, so the container of this getter covers both
	// sections without changing the fixture every other endpoint test shares.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	inventoryAt := int64(0x310) + 0x10020 + 505
	keyRowAt := inventoryAt + candidatesKeySectionAt
	binary.LittleEndian.PutUint32(data[keyRowAt:], candidatesStonebarbTearHandle)
	binary.LittleEndian.PutUint32(data[keyRowAt+4:], 1)
	binary.LittleEndian.PutUint32(data[keyRowAt+8:], 4)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("update fixture: %v", err)
	}

	engine := saveengine.New()
	loaded, err := engine.LoadSave(path, platform, "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	return engine, loaded.SaveSessionID, newPouchCatalog(t)
}

func TestGetEquipmentCandidatesFiltersEverySupportedSlotType(t *testing.T) {
	engine, saveSessionID, gameCatalog := loadEquipmentCandidatesFixture(t)

	tests := []struct {
		name         string
		slotType     string
		search       string
		pageSize     int
		wantKeys     []string
		wantOwned    bool
		wantTotalMin int
	}{
		{
			// The one owned record the catalog allows in a Quick Item slot is
			// offered with its owned identity and its stored quantity. The
			// Memory Stone beside it has no confirmed equipment capability and
			// the talisman has the wrong family, so neither is offered.
			name:      "quick item offers only the compatible owned record",
			slotType:  "quick_item",
			wantKeys:  []string{candidatesThrowingDaggerKey},
			wantOwned: true,
		},
		{
			// The same record is allowed in the Pouch slot, which proves the
			// filter reads the catalog's allowed slots and not only the family.
			name:      "pouch offers the same record",
			slotType:  "pouch",
			wantKeys:  []string{candidatesThrowingDaggerKey},
			wantOwned: true,
		},
		{
			name:      "talisman offers the owned talisman",
			slotType:  "talisman",
			wantKeys:  []string{candidatesMoonKey},
			wantOwned: true,
		},
		{
			// No owned weapon, armor or Crystal Tear exists in this container.
			name:     "hand slot offers nothing without a compatible owned record",
			slotType: "right_hand",
			wantKeys: []string{},
		},
		{
			name:     "armor slot offers nothing without a compatible owned record",
			slotType: "head",
			wantKeys: []string{},
		},
		{
			// The owned Crystal Tear lives in Inventory key, which is the second
			// section SetPhysickMixture accepts. Reading Inventory common alone
			// would offer nothing here and contradict the setter. The candidate
			// carries no owned identity: the Physick setter commits a catalog
			// reference.
			name:     "physick offers the Crystal Tear owned in Inventory key",
			slotType: "physick",
			wantKeys: []string{candidatesStonebarbTearKey},
		},
		{
			// Spells come from the catalog and never from the container, so the
			// served page is capped by pageSize while the total is not.
			name:         "spell memory pages the catalog",
			slotType:     "spell_memory",
			pageSize:     1,
			wantTotalMin: 2,
		},
		{
			name:     "search filters the owned candidates",
			slotType: "quick_item",
			search:   "no such item",
			wantKeys: []string{},
		},
		{
			name:      "search matches the candidate name",
			slotType:  "quick_item",
			search:    "throwing dag",
			wantKeys:  []string{candidatesThrowingDaggerKey},
			wantOwned: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := GetEquipmentCandidates(
				engine, gameCatalog, "safe", saveSessionID, 0,
				test.slotType, test.search, 0, test.pageSize)
			if err != nil {
				t.Fatalf("GetEquipmentCandidates: %v", err)
			}
			if result.SaveSessionID != saveSessionID || result.SaveRevision != "0" ||
				!result.Active || result.SafetyProfile != "safe" ||
				string(result.SlotType) != test.slotType {
				t.Fatalf("identity = %+v", result)
			}
			if test.wantTotalMin > 0 {
				if result.Total < test.wantTotalMin || len(result.Candidates) != test.pageSize {
					t.Fatalf("total = %d, served = %d", result.Total, len(result.Candidates))
				}
				for _, option := range result.Candidates {
					if option.MemorySlots <= 0 || option.OwnedItemID != "" {
						t.Errorf("spell candidate = %+v", option)
					}
				}
				return
			}
			if len(result.Candidates) != len(test.wantKeys) || result.Total != len(test.wantKeys) {
				t.Fatalf("candidates = %+v, want keys %v", result.Candidates, test.wantKeys)
			}
			for index, key := range test.wantKeys {
				option := result.Candidates[index]
				if option.Resource.Key != key || option.Name == "" {
					t.Errorf("candidate %d = %+v, want key %q", index, option, key)
				}
				if test.wantOwned && (option.OwnedItemID == "" || option.Quantity == 0) {
					t.Errorf("candidate %d = %+v, want an owned identity", index, option)
				}
				if !test.wantOwned && option.OwnedItemID != "" {
					t.Errorf("candidate %d = %+v, want no owned identity", index, option)
				}
				if option.MemorySlots != 0 {
					t.Errorf("candidate %d = %+v, want no memory cost", index, option)
				}
			}
		})
	}
}

func TestGetEquipmentCandidatesRejectsUnsupportedRequests(t *testing.T) {
	engine, saveSessionID, gameCatalog := loadEquipmentCandidatesFixture(t)

	tests := []struct {
		name     string
		profile  string
		slotType string
		page     int
		pageSize int
		want     string
	}{
		// Ammunition has no confirmed writer, so it must not be presented as an
		// editable slot through a candidate list either.
		{name: "arrow", profile: "safe", slotType: "arrow", want: "no equipment candidate contract"},
		{name: "bolt", profile: "safe", slotType: "bolt", want: "no equipment candidate contract"},
		{name: "unknown slot", profile: "safe", slotType: "hat", want: "no equipment candidate contract"},
		{name: "empty slot", profile: "safe", slotType: "", want: "no equipment candidate contract"},
		{name: "unknown profile", profile: "reckless", slotType: "talisman", want: "reckless"},
		{
			name: "negative page", profile: "safe", slotType: "talisman",
			page: -1, want: "page must not be negative",
		},
		{
			name: "negative page size", profile: "safe", slotType: "talisman",
			pageSize: -1, want: "pageSize must not be negative",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := GetEquipmentCandidates(
				engine, gameCatalog, test.profile, saveSessionID, 0,
				test.slotType, "", test.page, test.pageSize)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want it to contain %q", err, test.want)
			}
		})
	}
}

func TestGetEquipmentCandidatesRejectsMissingDependencies(t *testing.T) {
	if _, err := GetEquipmentCandidates(
		nil, newPouchCatalog(t), "safe", "session", 0, "talisman", "", 1, 10,
	); err == nil || err.Error() != "save engine is not available" {
		t.Fatalf("nil engine error = %v", err)
	}
	if _, err := GetEquipmentCandidates(
		saveengine.New(), nil, "safe", "session", 0, "talisman", "", 1, 10,
	); err == nil || err.Error() != "game catalog is not available" {
		t.Fatalf("nil catalog error = %v", err)
	}
}
