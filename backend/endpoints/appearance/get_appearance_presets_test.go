package appearance

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	catalogdata "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/data"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

// storedPresetCount is the number of presets presets/appearance.json holds.
const storedPresetCount = 20

// testManifest is the smallest manifest a catalog accepts. The getter reads no
// resource, so building the full prototype catalog would only slow the tests.
func testManifest() schema.Manifest {
	return schema.Manifest{
		SchemaVersion: schema.CurrentSchemaVersion,
		DataVersion:   "test",
		GameVersion:   "test",
		Sources: []schema.DataSource{{
			ID:       "test",
			Kind:     "test",
			Location: "backend/gamecatalog/data/presets/appearance.json",
			Version:  "test",
			Evidence: schema.EvidenceCurated,
		}},
	}
}

// newCatalog builds a resource-free catalog carrying the real stored presets.
func newCatalog(t *testing.T) *gamecatalog.Catalog {
	t.Helper()

	presets, err := gamecatalog.LoadAppearancePresets(catalogdata.Files())
	if err != nil {
		t.Fatalf("gamecatalog.LoadAppearancePresets: %v", err)
	}
	gameCatalog, err := gamecatalog.NewWithData(gamecatalog.CatalogData{
		Manifest:          testManifest(),
		AppearancePresets: presets,
	})
	if err != nil {
		t.Fatalf("gamecatalog.NewWithData: %v", err)
	}
	return gameCatalog
}

// newTaggedCatalog builds a small local catalog whose presets carry tags, since
// the stored data declares none.
func newTaggedCatalog(t *testing.T) *gamecatalog.Catalog {
	t.Helper()

	gameCatalog, err := gamecatalog.NewWithData(gamecatalog.CatalogData{
		Manifest: testManifest(),
		AppearancePresets: []gamecatalog.AppearancePreset{
			{ID: "alpha", Name: "Alpha", Image: "alpha.jpg", BodyType: 1, Tags: []string{"witcher", "male"}},
			{ID: "beta", Name: "Beta", Image: "beta.jpg", BodyType: 0, Tags: []string{"witcher"}},
			{ID: "gamma", Name: "Gamma", Image: "gamma.jpg", BodyType: 0, Tags: []string{}},
		},
	})
	if err != nil {
		t.Fatalf("gamecatalog.NewWithData: %v", err)
	}
	return gameCatalog
}

func idsOf(result GetAppearancePresetsResult) []string {
	ids := make([]string, len(result.Presets))
	for index, preset := range result.Presets {
		ids[index] = preset.ID
	}
	return ids
}

func TestGetAppearancePresetsReturnsEverySummaryInOrder(t *testing.T) {
	gameCatalog := newCatalog(t)

	result, err := GetAppearancePresets(gameCatalog, "", nil)
	if err != nil {
		t.Fatalf("GetAppearancePresets = %v, want nil", err)
	}
	if len(result.Presets) != storedPresetCount {
		t.Fatalf("len(Presets) = %d, want %d", len(result.Presets), storedPresetCount)
	}

	presets, err := gameCatalog.AppearancePresets()
	if err != nil {
		t.Fatalf("AppearancePresets: %v", err)
	}
	for index, preset := range presets {
		summary := result.Presets[index]
		if summary.ID != preset.ID || summary.Name != preset.Name || summary.Image != preset.Image {
			t.Fatalf("Presets[%d] = %#v, want the catalog order entry %#v", index, summary, preset)
		}
		if summary.Tags == nil {
			t.Fatalf("Presets[%d].Tags is nil, want a non-nil slice", index)
		}
	}
}

func TestGetAppearancePresetsMapsTheBodyType(t *testing.T) {
	result, err := GetAppearancePresets(newCatalog(t), "", nil)
	if err != nil {
		t.Fatalf("GetAppearancePresets = %v, want nil", err)
	}

	// geralt-of-rivia-the-witcher is stored as 1 and yennefer as 0.
	labels := map[string]string{}
	for _, summary := range result.Presets {
		labels[summary.ID] = summary.BodyType
	}
	if got := labels["geralt-of-rivia-the-witcher"]; got != "Type A" {
		t.Fatalf("geralt bodyType = %q, want Type A", got)
	}
	if got := labels["yennefer-sorceress-from-the-witcher"]; got != "Type B" {
		t.Fatalf("yennefer bodyType = %q, want Type B", got)
	}
	for id, label := range labels {
		if label != "Type A" && label != "Type B" {
			t.Fatalf("%s bodyType = %q, want Type A or Type B", id, label)
		}
	}
}

func TestGetAppearancePresetsSearchesIDAndNameCaseInsensitively(t *testing.T) {
	gameCatalog := newCatalog(t)

	// The id carries "the-witcher"; the name carries "Witcher" with a capital W.
	byID, err := GetAppearancePresets(gameCatalog, "GERALT-OF-RIVIA", nil)
	if err != nil {
		t.Fatalf("GetAppearancePresets = %v, want nil", err)
	}
	if got := idsOf(byID); len(got) != 1 || got[0] != "geralt-of-rivia-the-witcher" {
		t.Fatalf("ids = %#v, want only geralt-of-rivia-the-witcher", got)
	}

	byName, err := GetAppearancePresets(gameCatalog, "yennefer, sorceress", nil)
	if err != nil {
		t.Fatalf("GetAppearancePresets = %v, want nil", err)
	}
	if got := idsOf(byName); len(got) != 1 || got[0] != "yennefer-sorceress-from-the-witcher" {
		t.Fatalf("ids = %#v, want only yennefer-sorceress-from-the-witcher", got)
	}

	// A shared substring keeps the catalog order.
	shared, err := GetAppearancePresets(gameCatalog, "witcher", nil)
	if err != nil {
		t.Fatalf("GetAppearancePresets = %v, want nil", err)
	}
	if len(shared.Presets) < 2 {
		t.Fatalf("ids = %#v, want more than one witcher preset", idsOf(shared))
	}
	all, err := GetAppearancePresets(gameCatalog, "", nil)
	if err != nil {
		t.Fatalf("GetAppearancePresets = %v, want nil", err)
	}
	position := 0
	for _, summary := range all.Presets {
		if position < len(shared.Presets) && shared.Presets[position].ID == summary.ID {
			position++
		}
	}
	if position != len(shared.Presets) {
		t.Fatalf("filtered ids %#v do not keep the catalog order", idsOf(shared))
	}
}

// The search value is used exactly as supplied, so surrounding whitespace is
// part of the substring instead of being trimmed away.
func TestGetAppearancePresetsDoesNotTrimTheSearch(t *testing.T) {
	result, err := GetAppearancePresets(newCatalog(t), " geralt", nil)
	if err != nil {
		t.Fatalf("GetAppearancePresets = %v, want nil", err)
	}
	if len(result.Presets) != 0 {
		t.Fatalf("ids = %#v, want no match for a padded search", idsOf(result))
	}
}

func TestGetAppearancePresetsReturnsAnEmptyNonNilListWithoutMatches(t *testing.T) {
	result, err := GetAppearancePresets(newCatalog(t), "no-such-preset", nil)
	if err != nil {
		t.Fatalf("GetAppearancePresets = %v, want nil", err)
	}
	if result.Presets == nil {
		t.Fatal("Presets is nil, want an empty, non-nil slice")
	}
	if len(result.Presets) != 0 {
		t.Fatalf("ids = %#v, want no match", idsOf(result))
	}

	// The stored presets declare no tags, so any tag filter empties the list.
	tagged, err := GetAppearancePresets(newCatalog(t), "", []string{"witcher"})
	if err != nil {
		t.Fatalf("GetAppearancePresets = %v, want nil", err)
	}
	if tagged.Presets == nil || len(tagged.Presets) != 0 {
		t.Fatalf("Presets = %#v, want an empty, non-nil slice", tagged.Presets)
	}
}

func TestGetAppearancePresetsFiltersTagsWithAndSemantics(t *testing.T) {
	gameCatalog := newTaggedCatalog(t)

	cases := []struct {
		name string
		tags []string
		want []string
	}{
		{name: "nil does not filter", tags: nil, want: []string{"alpha", "beta", "gamma"}},
		{name: "empty does not filter", tags: []string{}, want: []string{"alpha", "beta", "gamma"}},
		{name: "one tag", tags: []string{"witcher"}, want: []string{"alpha", "beta"}},
		{name: "every tag must match", tags: []string{"witcher", "male"}, want: []string{"alpha"}},
		{name: "unknown tag", tags: []string{"witcher", "elf"}, want: []string{}},
		{name: "case sensitive", tags: []string{"Witcher"}, want: []string{}},
		{name: "not trimmed", tags: []string{" witcher"}, want: []string{}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := GetAppearancePresets(gameCatalog, "", testCase.tags)
			if err != nil {
				t.Fatalf("GetAppearancePresets = %v, want nil", err)
			}
			got := idsOf(result)
			if len(got) != len(testCase.want) {
				t.Fatalf("ids = %#v, want %#v", got, testCase.want)
			}
			for index, want := range testCase.want {
				if got[index] != want {
					t.Fatalf("ids = %#v, want %#v", got, testCase.want)
				}
			}
		})
	}
}

func TestGetAppearancePresetsRejectsAnEmptyTag(t *testing.T) {
	gameCatalog := newTaggedCatalog(t)

	for _, tags := range [][]string{{""}, {"witcher", ""}, {"", "witcher"}} {
		result, err := GetAppearancePresets(gameCatalog, "", tags)
		if err == nil {
			t.Fatalf("GetAppearancePresets(%#v) = nil error, want a rejection", tags)
		}
		if !strings.Contains(err.Error(), "must not be empty") {
			t.Fatalf("error = %q, want an empty-tag rejection", err.Error())
		}
		if len(result.Presets) != 0 {
			t.Fatalf("Presets = %#v, want an empty result", result.Presets)
		}
	}
}

func TestGetAppearancePresetsRequiresACatalogWithPresets(t *testing.T) {
	if _, err := GetAppearancePresets(nil, "", nil); err == nil {
		t.Fatal("GetAppearancePresets(nil) = nil error, want a rejection")
	} else if err.Error() != "game catalog is not loaded" {
		t.Fatalf("error = %q, want %q", err.Error(), "game catalog is not loaded")
	}

	withoutPresets, err := gamecatalog.New(testManifest(), nil)
	if err != nil {
		t.Fatalf("gamecatalog.New: %v", err)
	}
	if _, err := GetAppearancePresets(withoutPresets, "", nil); err == nil {
		t.Fatal("GetAppearancePresets without presets = nil error, want a rejection")
	} else if err.Error() != "appearance presets are not loaded" {
		t.Fatalf("error = %q, want %q", err.Error(), "appearance presets are not loaded")
	}
}

// Every call builds its own result, so mutating one must not affect the next.
func TestGetAppearancePresetsBuildsAnIndependentResultPerCall(t *testing.T) {
	gameCatalog := newTaggedCatalog(t)

	first, err := GetAppearancePresets(gameCatalog, "", nil)
	if err != nil {
		t.Fatalf("GetAppearancePresets = %v, want nil", err)
	}
	want := first.Presets[0]
	first.Presets[0].ID = "mutated"
	first.Presets[0].BodyType = "mutated"
	first.Presets[0].Tags[0] = "mutated"

	second, err := GetAppearancePresets(gameCatalog, "", nil)
	if err != nil {
		t.Fatalf("GetAppearancePresets = %v, want nil", err)
	}
	if second.Presets[0].ID != want.ID || second.Presets[0].BodyType != want.BodyType {
		t.Fatalf("Presets[0] = %#v, want %#v", second.Presets[0], want)
	}
	if second.Presets[0].Tags[0] != "witcher" {
		t.Fatalf("Presets[0].Tags = %#v, want the stored tags", second.Presets[0].Tags)
	}
}

// The list metadata must not leak the appearance configuration the mutating
// endpoint applies.
func TestGetAppearancePresetsResultCarriesOnlyListMetadata(t *testing.T) {
	result, err := GetAppearancePresets(newCatalog(t), "", nil)
	if err != nil {
		t.Fatalf("GetAppearancePresets = %v, want nil", err)
	}
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}

	var decoded struct {
		Presets []map[string]json.RawMessage `json:"presets"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if len(decoded.Presets) != storedPresetCount {
		t.Fatalf("len(presets) = %d, want %d", len(decoded.Presets), storedPresetCount)
	}
	allowed := map[string]struct{}{"id": {}, "name": {}, "image": {}, "bodyType": {}, "tags": {}}
	for index, preset := range decoded.Presets {
		if len(preset) != len(allowed) {
			t.Fatalf("presets[%d] has fields %#v, want exactly %d", index, preset, len(allowed))
		}
		for field := range preset {
			if _, ok := allowed[field]; !ok {
				t.Fatalf("presets[%d] carries the forbidden field %q", index, field)
			}
		}
	}
	for _, forbidden := range []string{"voiceType", "faceModel", "hairModel", "faceShape", "body", "skin"} {
		if strings.Contains(string(raw), `"`+forbidden+`":`) {
			t.Fatalf("the result JSON carries %q", forbidden)
		}
	}
}
