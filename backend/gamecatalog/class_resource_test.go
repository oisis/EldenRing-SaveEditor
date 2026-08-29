package gamecatalog_test

import (
	"strings"
	"sync"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/catalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	catalogdata "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/data"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/loader"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

var (
	storedCatalog     *gamecatalog.Catalog
	storedCatalogErr  error
	storedCatalogOnce sync.Once
)

func newRealCatalog(t *testing.T) *gamecatalog.Catalog {
	t.Helper()

	storedCatalogOnce.Do(func() {
		data, err := loader.LoadFS(catalogdata.Files())
		if err != nil {
			storedCatalogErr = err
			return
		}
		storedCatalog, storedCatalogErr = gamecatalog.New(data.Manifest, data.Resources())
	})
	if storedCatalogErr != nil {
		t.Fatalf("build stored catalog: %v", storedCatalogErr)
	}
	return storedCatalog
}

func TestAllTwelveClassesLoadWithExactConfirmedMapping(t *testing.T) {
	cat := newRealCatalog(t)

	// level is the CharaInitParam soulLv fact of each class: for 0..9 confirmed
	// against a native vanilla save holding all ten freshly created classes, for
	// the Regulation 1.17 classes 10 and 11 read from CharaInitParam rows 3010
	// and 3011. It is a fact in its own right and is deliberately not asserted as
	// sum(attributes)-79: that formula belongs to SetCharacterStats, not to this
	// document.
	expectedClasses := map[string]struct {
		id           uint32
		name         string
		level        uint32
		vigor        uint32
		mind         uint32
		endurance    uint32
		strength     uint32
		dexterity    uint32
		intelligence uint32
		faith        uint32
		arcane       uint32
	}{
		"0": {0, "Vagabond", 9, 15, 10, 11, 14, 13, 9, 9, 7},
		"1": {1, "Warrior", 8, 11, 12, 11, 10, 16, 10, 8, 9},
		"2": {2, "Hero", 7, 14, 9, 12, 16, 9, 7, 8, 11},
		"3": {3, "Bandit", 5, 10, 11, 10, 9, 13, 9, 8, 14},
		"4": {4, "Astrologer", 6, 9, 15, 9, 8, 12, 16, 7, 9},
		"5": {5, "Prophet", 7, 10, 14, 8, 11, 10, 7, 16, 10},
		"6": {6, "Confessor", 10, 10, 13, 10, 12, 12, 9, 14, 9},
		"7": {7, "Samurai", 9, 12, 11, 13, 12, 15, 9, 8, 8},
		"8": {8, "Prisoner", 9, 11, 12, 11, 11, 14, 14, 6, 9},
		"9": {9, "Wretch", 1, 10, 10, 10, 10, 10, 10, 10, 10},
		// Regulation 1.17, CharaInitParam rows 3010 and 3011, names from the
		// menu_dlc02 GR_MenuText captions 297140 and 297141.
		"10": {10, "Idus Knight", 7, 10, 12, 11, 13, 15, 8, 11, 6},
		"11": {11, "Heavy Knight", 10, 14, 8, 17, 15, 11, 7, 8, 9},
	}

	for key, expected := range expectedClasses {
		res, err := catalog.GetResource(cat, string(schema.ResourceKindClass), key)
		if err != nil {
			t.Fatalf("GetResource(class, %q): %v", key, err)
		}
		if res.Resource.Kind != schema.ResourceKindClass {
			t.Fatalf("key %q: kind = %q, want %q", key, res.Resource.Kind, schema.ResourceKindClass)
		}
		if res.Resource.Class == nil {
			t.Fatalf("key %q: class document is nil", key)
		}
		doc := res.Resource.Class
		if !doc.StartingClassID.Known || doc.StartingClassID.Value != expected.id {
			t.Fatalf("key %q: startingClassID = %v, want %d", key, doc.StartingClassID, expected.id)
		}
		if !doc.Name.Known || doc.Name.Value != expected.name {
			t.Fatalf("key %q: name = %v, want %q", key, doc.Name, expected.name)
		}
		if !doc.Level.Known || doc.Level.Value != expected.level {
			t.Fatalf("key %q: level = %v, want %d", key, doc.Level, expected.level)
		}
		if doc.Level.Provenance.Source == "" {
			t.Fatalf("key %q: level provenance source is empty", key)
		}
		if !doc.Vigor.Known || doc.Vigor.Value != expected.vigor {
			t.Fatalf("key %q: vigor = %v, want %d", key, doc.Vigor, expected.vigor)
		}
		if !doc.Mind.Known || doc.Mind.Value != expected.mind {
			t.Fatalf("key %q: mind = %v, want %d", key, doc.Mind, expected.mind)
		}
		if !doc.Endurance.Known || doc.Endurance.Value != expected.endurance {
			t.Fatalf("key %q: endurance = %v, want %d", key, doc.Endurance, expected.endurance)
		}
		if !doc.Strength.Known || doc.Strength.Value != expected.strength {
			t.Fatalf("key %q: strength = %v, want %d", key, doc.Strength, expected.strength)
		}
		if !doc.Dexterity.Known || doc.Dexterity.Value != expected.dexterity {
			t.Fatalf("key %q: dexterity = %v, want %d", key, doc.Dexterity, expected.dexterity)
		}
		if !doc.Intelligence.Known || doc.Intelligence.Value != expected.intelligence {
			t.Fatalf("key %q: intelligence = %v, want %d", key, doc.Intelligence, expected.intelligence)
		}
		if !doc.Faith.Known || doc.Faith.Value != expected.faith {
			t.Fatalf("key %q: faith = %v, want %d", key, doc.Faith, expected.faith)
		}
		if !doc.Arcane.Known || doc.Arcane.Value != expected.arcane {
			t.Fatalf("key %q: arcane = %v, want %d", key, doc.Arcane, expected.arcane)
		}
	}

	// Explicitly assert the critical 6 Confessor, 7 Samurai, 8 Prisoner mappings
	// so that create-character menu display order (297130-297139) is never accidentally used.
	confessor, err := catalog.GetResource(cat, string(schema.ResourceKindClass), "6")
	if err != nil || confessor.Resource.Class.Name.Value != "Confessor" ||
		confessor.Resource.Class.StartingClassID.Value != 6 ||
		confessor.Resource.Class.Level.Value != 10 {
		t.Fatalf("class 6 mapping failed: got %+v, want Confessor (ID 6, level 10)", confessor)
	}

	samurai, err := catalog.GetResource(cat, string(schema.ResourceKindClass), "7")
	if err != nil || samurai.Resource.Class.Name.Value != "Samurai" ||
		samurai.Resource.Class.StartingClassID.Value != 7 ||
		samurai.Resource.Class.Level.Value != 9 {
		t.Fatalf("class 7 mapping failed: got %+v, want Samurai (ID 7, level 9)", samurai)
	}

	prisoner, err := catalog.GetResource(cat, string(schema.ResourceKindClass), "8")
	if err != nil || prisoner.Resource.Class.Name.Value != "Prisoner" ||
		prisoner.Resource.Class.StartingClassID.Value != 8 ||
		prisoner.Resource.Class.Level.Value != 9 {
		t.Fatalf("class 8 mapping failed: got %+v, want Prisoner (ID 8, level 9)", prisoner)
	}

	// The same explicit pin for the two Regulation 1.17 classes: their level is
	// the CharaInitParam soulLv of rows 3010 and 3011, never the create-character
	// menu order of CharMakeMenuListItemParam rows 100210 and 100211.
	idusKnight, err := catalog.GetResource(cat, string(schema.ResourceKindClass), "10")
	if err != nil || idusKnight.Resource.Class.Name.Value != "Idus Knight" ||
		idusKnight.Resource.Class.StartingClassID.Value != 10 ||
		idusKnight.Resource.Class.Level.Value != 7 {
		t.Fatalf("class 10 mapping failed: got %+v, want Idus Knight (ID 10, level 7)", idusKnight)
	}

	heavyKnight, err := catalog.GetResource(cat, string(schema.ResourceKindClass), "11")
	if err != nil || heavyKnight.Resource.Class.Name.Value != "Heavy Knight" ||
		heavyKnight.Resource.Class.StartingClassID.Value != 11 ||
		heavyKnight.Resource.Class.Level.Value != 10 {
		t.Fatalf("class 11 mapping failed: got %+v, want Heavy Knight (ID 11, level 10)", heavyKnight)
	}
}

// TestRegulation117ClassNamesCiteTheirTextSource pins where the two names of the
// Regulation 1.17 classes actually come from. The name is FMG text, not a
// CharaInitParam column: a provenance naming CharaInitParam and its row 3010 or
// 3011 would claim the regulation carries the English string, which it does not.
// The document must cite the GR_MenuText caption row that holds it, so the exact
// table, row and field are asserted here rather than only the source ID.
func TestRegulation117ClassNamesCiteTheirTextSource(t *testing.T) {
	cat := newRealCatalog(t)

	for _, want := range []struct {
		key   string
		name  string
		row   string
		menu  string
		level uint32
	}{
		{key: "10", name: "Idus Knight", row: "297140", menu: "100210", level: 7},
		{key: "11", name: "Heavy Knight", row: "297141", menu: "100211", level: 10},
	} {
		res, err := catalog.GetResource(cat, string(schema.ResourceKindClass), want.key)
		if err != nil {
			t.Fatalf("GetResource(class, %q): %v", want.key, err)
		}
		doc := res.Resource.Class
		if doc == nil {
			t.Fatalf("key %q: class document is nil", want.key)
		}

		name := doc.Name
		if !name.Known || name.Value != want.name {
			t.Fatalf("key %q: name = %v, want %q", want.key, name, want.name)
		}
		if name.Provenance.Source != "game_text_gr_menu_text_dlc02" {
			t.Errorf("key %q: name provenance source = %q, want game_text_gr_menu_text_dlc02",
				want.key, name.Provenance.Source)
		}
		if name.Provenance.Table != "GR_MenuText" {
			t.Errorf("key %q: name provenance table = %q, want GR_MenuText",
				want.key, name.Provenance.Table)
		}
		if name.Provenance.Row != want.row {
			t.Errorf("key %q: name provenance row = %q, want the caption row %q",
				want.key, name.Provenance.Row, want.row)
		}
		if name.Provenance.Field != "Text" {
			t.Errorf("key %q: name provenance field = %q, want Text",
				want.key, name.Provenance.Field)
		}
		if !strings.Contains(name.Provenance.Method, want.menu) {
			t.Errorf("key %q: name provenance method = %q, want it to name the "+
				"CharMakeMenuListItemParam row %q that selects the caption",
				want.key, name.Provenance.Method, want.menu)
		}

		// The numeric facts keep citing CharaInitParam, which is where they really
		// live. Pinning one of them here proves the fix did not move every fact of
		// the document onto the text source.
		if !doc.Level.Known || doc.Level.Value != want.level {
			t.Fatalf("key %q: level = %v, want %d", want.key, doc.Level, want.level)
		}
		if doc.Level.Provenance.Source != "regulation_chara_init_param" ||
			doc.Level.Provenance.Table != "CharaInitParam" ||
			doc.Level.Provenance.Field != "soulLv" {
			t.Errorf("key %q: level provenance = %+v, want the CharaInitParam soulLv fact",
				want.key, doc.Level.Provenance)
		}
	}
}

func TestGetResourcesReturnsTwelveClasses(t *testing.T) {
	cat := newRealCatalog(t)

	result, err := catalog.GetResources(cat, string(schema.ResourceKindClass), "", "", "", "", 0, 0)
	if err != nil {
		t.Fatalf("GetResources(class): %v", err)
	}
	if result.Total != 12 {
		t.Fatalf("class total = %d, want 12", result.Total)
	}
	if len(result.Resources) != 12 {
		t.Fatalf("class resources length = %d, want 12", len(result.Resources))
	}

	expectedOrder := []struct {
		key  string
		name string
	}{
		{"0", "Vagabond"},
		{"1", "Warrior"},
		{"2", "Hero"},
		{"3", "Bandit"},
		{"4", "Astrologer"},
		{"5", "Prophet"},
		{"6", "Confessor"},
		{"7", "Samurai"},
		{"8", "Prisoner"},
		{"9", "Wretch"},
		{"10", "Idus Knight"},
		{"11", "Heavy Knight"},
	}

	for i, expected := range expectedOrder {
		if result.Resources[i].Key != expected.key || result.Resources[i].Name != expected.name {
			t.Errorf("class[%d] = %+v, want key %q name %q", i, result.Resources[i], expected.key, expected.name)
		}
	}
}

func TestGetResourceReturnsIndependentClassDocument(t *testing.T) {
	cat := newRealCatalog(t)

	result, err := catalog.GetResource(cat, string(schema.ResourceKindClass), "0")
	if err != nil {
		t.Fatalf("GetResource(class, 0): %v", err)
	}
	if result.Resource.Class.Name.Value != "Vagabond" {
		t.Fatalf("name = %q, want Vagabond", result.Resource.Class.Name.Value)
	}
	if !result.Resource.Class.Level.Known || result.Resource.Class.Level.Value != 9 {
		t.Fatalf("level = %v, want the known value 9", result.Resource.Class.Level)
	}

	// Mutate to verify deep-copy immutability.
	result.Resource.Class.Name.Value = "Mutated"
	result.Resource.Class.StartingClassID.Value = 999
	result.Resource.Class.Level.Value = 999

	again, err := catalog.GetResource(cat, string(schema.ResourceKindClass), "0")
	if err != nil {
		t.Fatalf("GetResource again: %v", err)
	}
	if again.Resource.Class.Name.Value != "Vagabond" {
		t.Fatalf("catalog class was mutated: %q", again.Resource.Class.Name.Value)
	}
	if again.Resource.Class.StartingClassID.Value != 0 {
		t.Fatalf("catalog class ID was mutated: %d", again.Resource.Class.StartingClassID.Value)
	}
	if again.Resource.Class.Level.Value != 9 {
		t.Fatalf("catalog class level was mutated: %d", again.Resource.Class.Level.Value)
	}
}

func TestValidateClassResourceRejectsMalformed(t *testing.T) {
	data, err := loader.LoadFS(catalogdata.Files())
	if err != nil {
		t.Fatalf("load embedded data: %v", err)
	}
	sources := make(map[schema.SourceID]struct{}, len(data.Manifest.Sources))
	for _, source := range data.Manifest.Sources {
		sources[source.ID] = struct{}{}
	}
	var validProvenance schema.Provenance
	for _, res := range data.Resources() {
		if res.Class != nil {
			validProvenance = res.Class.StartingClassID.Provenance
			break
		}
	}

	validClassDoc := func() *schema.ClassDocument {
		return &schema.ClassDocument{
			StartingClassID: schema.Fact[uint32]{Known: true, Value: 0, Provenance: validProvenance},
			Name:            schema.Fact[string]{Known: true, Value: "Vagabond", Provenance: validProvenance},
			Level:           schema.Fact[uint32]{Known: true, Value: 9, Provenance: validProvenance},
			Vigor:           schema.Fact[uint32]{Known: true, Value: 15, Provenance: validProvenance},
			Mind:            schema.Fact[uint32]{Known: true, Value: 10, Provenance: validProvenance},
			Endurance:       schema.Fact[uint32]{Known: true, Value: 11, Provenance: validProvenance},
			Strength:        schema.Fact[uint32]{Known: true, Value: 14, Provenance: validProvenance},
			Dexterity:       schema.Fact[uint32]{Known: true, Value: 13, Provenance: validProvenance},
			Intelligence:    schema.Fact[uint32]{Known: true, Value: 9, Provenance: validProvenance},
			Faith:           schema.Fact[uint32]{Known: true, Value: 9, Provenance: validProvenance},
			Arcane:          schema.Fact[uint32]{Known: true, Value: 7, Provenance: validProvenance},
		}
	}

	testCases := []struct {
		name     string
		resource schema.Resource
	}{
		{
			name: "key outside 0-11",
			resource: schema.Resource{
				Key:   "12",
				Kind:  schema.ResourceKindClass,
				Class: validClassDoc(),
			},
		},
		{
			name: "key with a leading zero",
			resource: schema.Resource{
				Key:   "01",
				Kind:  schema.ResourceKindClass,
				Class: validClassDoc(),
			},
		},
		{
			name: "alpha key",
			resource: schema.Resource{
				Key:   "a",
				Kind:  schema.ResourceKindClass,
				Class: validClassDoc(),
			},
		},
		{
			name: "StartingClassID disagrees with key",
			resource: func() schema.Resource {
				doc := validClassDoc()
				doc.StartingClassID.Value = 1
				return schema.Resource{
					Key:   "0",
					Kind:  schema.ResourceKindClass,
					Class: doc,
				}
			}(),
		},
		{
			name: "empty name",
			resource: func() schema.Resource {
				doc := validClassDoc()
				doc.Name.Value = ""
				return schema.Resource{
					Key:   "0",
					Kind:  schema.ResourceKindClass,
					Class: doc,
				}
			}(),
		},
		{
			name: "unknown name",
			resource: func() schema.Resource {
				doc := validClassDoc()
				doc.Name.Known = false
				return schema.Resource{
					Key:   "0",
					Kind:  schema.ResourceKindClass,
					Class: doc,
				}
			}(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if err := schema.ValidateResource(tc.resource, sources); err == nil {
				t.Fatalf("expected error for %s, got nil", tc.name)
			}
		})
	}
}
