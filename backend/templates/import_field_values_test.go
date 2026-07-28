package templates

import (
	"encoding/json"
	"testing"

	"gopkg.in/yaml.v3"
)

func sptr(s string) *string { return &s }

// fullV2ProfileStatsTemplate builds a valid v2 template that ships every
// profile field and all eight stats. clearCount / scadutreeBlessing /
// shadowRealmBlessing / talismanSlots are set to 0 on purpose so the
// "zero is a present value, not an absent field" invariant is exercised.
func fullV2ProfileStatsTemplate() *BuildTemplate {
	return &BuildTemplate{
		Schema:    SchemaKey,
		Version:   2,
		CreatedAt: "2026-07-28T12:00:00Z",
		Selection: &TemplateSelection{
			Profile: &SectionSelection{All: true},
			Stats:   &SectionSelection{All: true},
		},
		Sections: TemplateSections{
			Profile: &ProfileSection{
				Name:                sptr("Tarnished"),
				Level:               u32p(129),
				Runes:               u32p(5337),
				SoulMemory:          u32p(0),
				Class:               sptr("Vagabond"),
				ClearCount:          u32p(0),
				ScadutreeBlessing:   u8p(0),
				ShadowRealmBlessing: u8p(0),
				TalismanSlots:       u8p(0),
			},
			Stats: &StatsSection{
				Vigor:        u32p(60),
				Mind:         u32p(10),
				Endurance:    u32p(25),
				Strength:     u32p(50),
				Dexterity:    u32p(12),
				Intelligence: u32p(9),
				Faith:        u32p(8),
				Arcane:       u32p(7),
			},
		},
	}
}

func fvMap(pairs []FieldValue) map[string]string {
	m := make(map[string]string, len(pairs))
	for _, p := range pairs {
		m[p.Key] = p.Value
	}
	return m
}

func TestPreviewSummary_ProfileFieldValues_ExactAndOrdered(t *testing.T) {
	tpl := fullV2ProfileStatsTemplate()
	rep := PreviewBuildTemplateImport(tpl, ImportPreviewOptions{Mode: "append"})
	if !rep.OK {
		t.Fatalf("expected OK report, got errors: %+v", rep.Errors)
	}

	// Deterministic character-sheet order.
	wantOrder := []string{
		"name", "level", "runes", "soulMemory", "class",
		"clearCount", "scadutreeBlessing", "shadowRealmBlessing", "talismanSlots",
	}
	if len(rep.Summary.ProfileFieldValues) != len(wantOrder) {
		t.Fatalf("ProfileFieldValues len=%d want=%d (%+v)",
			len(rep.Summary.ProfileFieldValues), len(wantOrder), rep.Summary.ProfileFieldValues)
	}
	for i, key := range wantOrder {
		if rep.Summary.ProfileFieldValues[i].Key != key {
			t.Errorf("ProfileFieldValues[%d].Key = %q, want %q", i, rep.Summary.ProfileFieldValues[i].Key, key)
		}
	}

	want := map[string]string{
		"name": "Tarnished", "level": "129", "runes": "5337", "soulMemory": "0",
		"class": "Vagabond", "clearCount": "0", "scadutreeBlessing": "0",
		"shadowRealmBlessing": "0", "talismanSlots": "0",
	}
	got := fvMap(rep.Summary.ProfileFieldValues)
	for k, v := range want {
		if got[k] != v {
			t.Errorf("profile %q = %q, want %q", k, got[k], v)
		}
	}
}

func TestPreviewSummary_StatFieldValues_ExactAndOrdered(t *testing.T) {
	tpl := fullV2ProfileStatsTemplate()
	rep := PreviewBuildTemplateImport(tpl, ImportPreviewOptions{Mode: "append"})

	wantOrder := []string{
		"vigor", "mind", "endurance", "strength",
		"dexterity", "intelligence", "faith", "arcane",
	}
	if len(rep.Summary.StatFieldValues) != len(wantOrder) {
		t.Fatalf("StatFieldValues len=%d want=%d", len(rep.Summary.StatFieldValues), len(wantOrder))
	}
	for i, key := range wantOrder {
		if rep.Summary.StatFieldValues[i].Key != key {
			t.Errorf("StatFieldValues[%d].Key = %q, want %q", i, rep.Summary.StatFieldValues[i].Key, key)
		}
	}
	want := map[string]string{
		"vigor": "60", "mind": "10", "endurance": "25", "strength": "50",
		"dexterity": "12", "intelligence": "9", "faith": "8", "arcane": "7",
	}
	got := fvMap(rep.Summary.StatFieldValues)
	for k, v := range want {
		if got[k] != v {
			t.Errorf("stat %q = %q, want %q", k, got[k], v)
		}
	}
}

// TestPreviewSummary_FieldValues_JSONRoundTrip proves values survive a
// full JSON marshal → parse → preview cycle unchanged (both the profile
// zeros and every stat).
func TestPreviewSummary_FieldValues_JSONRoundTrip(t *testing.T) {
	data, err := json.Marshal(fullV2ProfileStatsTemplate())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	tpl, err := ParseBuildTemplateJSON(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	rep := PreviewBuildTemplateImport(tpl, ImportPreviewOptions{Mode: "append"})
	if fvMap(rep.Summary.ProfileFieldValues)["level"] != "129" ||
		fvMap(rep.Summary.ProfileFieldValues)["runes"] != "5337" ||
		fvMap(rep.Summary.ProfileFieldValues)["soulMemory"] != "0" {
		t.Errorf("JSON round-trip lost profile values: %+v", rep.Summary.ProfileFieldValues)
	}
	if fvMap(rep.Summary.StatFieldValues)["vigor"] != "60" ||
		fvMap(rep.Summary.StatFieldValues)["strength"] != "50" {
		t.Errorf("JSON round-trip lost stat values: %+v", rep.Summary.StatFieldValues)
	}
}

// TestPreviewSummary_FieldValues_YAMLRoundTrip mirrors the JSON test for
// the YAML codec used by the file/URL import paths.
func TestPreviewSummary_FieldValues_YAMLRoundTrip(t *testing.T) {
	data, err := yaml.Marshal(fullV2ProfileStatsTemplate())
	if err != nil {
		t.Fatalf("yaml marshal: %v", err)
	}
	var tpl BuildTemplate
	if err := yaml.Unmarshal(data, &tpl); err != nil {
		t.Fatalf("yaml unmarshal: %v", err)
	}
	rep := PreviewBuildTemplateImport(&tpl, ImportPreviewOptions{Mode: "append"})
	if fvMap(rep.Summary.ProfileFieldValues)["talismanSlots"] != "0" {
		t.Errorf("YAML round-trip lost talismanSlots=0: %+v", rep.Summary.ProfileFieldValues)
	}
	if fvMap(rep.Summary.StatFieldValues)["arcane"] != "7" {
		t.Errorf("YAML round-trip lost arcane: %+v", rep.Summary.StatFieldValues)
	}
}

// TestPreviewSummary_FieldValues_UnselectedFieldsAbsent proves a template
// that ships only a subset of fields reports exactly that subset — no
// phantom zero-valued entries for the omitted fields.
func TestPreviewSummary_FieldValues_UnselectedFieldsAbsent(t *testing.T) {
	tpl := &BuildTemplate{
		Schema:    SchemaKey,
		Version:   2,
		CreatedAt: "2026-07-28T12:00:00Z",
		Selection: &TemplateSelection{
			Profile: &SectionSelection{Fields: map[string]bool{"level": true}},
			Stats:   &SectionSelection{Fields: map[string]bool{"vigor": true}},
		},
		Sections: TemplateSections{
			Profile: &ProfileSection{Level: u32p(150)},
			Stats:   &StatsSection{Vigor: u32p(99)},
		},
	}
	rep := PreviewBuildTemplateImport(tpl, ImportPreviewOptions{Mode: "append"})
	if len(rep.Summary.ProfileFieldValues) != 1 || rep.Summary.ProfileFieldValues[0].Key != "level" {
		t.Errorf("expected only level, got %+v", rep.Summary.ProfileFieldValues)
	}
	if len(rep.Summary.StatFieldValues) != 1 || rep.Summary.StatFieldValues[0].Key != "vigor" {
		t.Errorf("expected only vigor, got %+v", rep.Summary.StatFieldValues)
	}
}

// TestPreviewSummary_FieldValues_LegacyReportUnaffected proves the
// name-only lists (ProfileFieldsPresent / StatFieldsPresent) are still
// populated exactly as before, so older consumers keep working.
func TestPreviewSummary_FieldValues_LegacyReportUnaffected(t *testing.T) {
	rep := PreviewBuildTemplateImport(fullV2ProfileStatsTemplate(), ImportPreviewOptions{Mode: "append"})
	if len(rep.Summary.ProfileFieldsPresent) != 9 {
		t.Errorf("ProfileFieldsPresent = %v, want 9 entries", rep.Summary.ProfileFieldsPresent)
	}
	if len(rep.Summary.StatFieldsPresent) != 8 {
		t.Errorf("StatFieldsPresent = %v, want 8 entries", rep.Summary.StatFieldsPresent)
	}
}
