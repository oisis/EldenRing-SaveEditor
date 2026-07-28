package application

import (
	"reflect"
	"strings"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/templates"
	"github.com/oisis/EldenRing-SaveForge/backend/vm"
)

func makeDeriveFixture() *App {
	app := applyV2Fixture()
	// profileStatsFixture predates class-minimum validation and carries
	// Dexterity 12 for a Vagabond. Real characters cannot be below their
	// starting-class minimum, so normalize this synthetic fixture to 13.
	app.save.Slots[0].Player.Dexterity = 13
	return app
}

func TestApplyBuildTemplateV2_DerivesLevelAndIgnoresTemplateLevel(t *testing.T) {
	app := makeDeriveFixture()
	jsonText := makeV2Template(t, app, `{"profile":{"level":true},"stats":{"vigor":true}}`, func(tpl *templates.BuildTemplate) {
		tpl.Sections.Profile.Level = u32(200)
		tpl.Sections.Stats.Vigor = u32(30)
	})

	res, err := app.ApplyBuildTemplateV2ToCharacterJSON(0, jsonText, ApplyTemplateV2Options{
		DeriveLevelFromStats: true,
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !res.Applied {
		t.Fatalf("Applied=false: %+v", res.Preview.Errors)
	}

	player := app.save.Slots[0].Player
	const wantLevel = uint32(38) // 30+15+18+16+13+9+9+7 - 79
	if player.Level != wantLevel {
		t.Fatalf("Level=%d, want derived %d (template requested 200)", player.Level, wantLevel)
	}
	if player.SoulMemory < vm.MinimumSoulMemoryForLevel(wantLevel) {
		t.Fatalf("SoulMemory=%d below derived-level minimum", player.SoulMemory)
	}
	if !containsString(res.AppliedFields, "profile.level") {
		t.Fatalf("AppliedFields=%v, want profile.level", res.AppliedFields)
	}
}

func TestApplyBuildTemplateV2_ClassOverrideZeroIsValid(t *testing.T) {
	app := makeDeriveFixture()
	name := "Class Zero"
	jsonText := makeV2Template(t, app, `{"profile":{"name":true}}`, func(tpl *templates.BuildTemplate) {
		tpl.Sections.Profile.Name = &name
	})
	classID := uint8(0)

	res, err := app.ApplyBuildTemplateV2ToCharacterJSON(0, jsonText, ApplyTemplateV2Options{
		DeriveLevelFromStats: true,
		ClassOverride:        &ClassOverride{ClassID: classID},
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !res.Applied {
		t.Fatalf("Applied=false: %+v", res.Preview.Errors)
	}
	if got := app.save.Slots[0].Player.Class; got != classID {
		t.Fatalf("Class=%d, want %d", got, classID)
	}
	if !containsString(res.AppliedFields, "profile.class") {
		t.Fatalf("AppliedFields=%v, want profile.class", res.AppliedFields)
	}
	if containsString(res.SkippedFields, "profile.class") {
		t.Fatalf("SkippedFields=%v unexpectedly contains profile.class", res.SkippedFields)
	}
}

func TestApplyBuildTemplateV2_ClassOverrideWithRaisedMinimumStats(t *testing.T) {
	app := makeDeriveFixture()
	jsonText := makeV2Template(t, app, `{"stats":{"dexterity":true,"intelligence":true,"arcane":true}}`, func(tpl *templates.BuildTemplate) {
		tpl.Sections.Stats.Dexterity = u32(16)
		tpl.Sections.Stats.Intelligence = u32(10)
		tpl.Sections.Stats.Arcane = u32(9)
	})

	res, err := app.ApplyBuildTemplateV2ToCharacterJSON(0, jsonText, ApplyTemplateV2Options{
		DeriveLevelFromStats: true,
		ClassOverride:        &ClassOverride{ClassID: 1}, // Warrior
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !res.Applied {
		t.Fatalf("Applied=false: %+v", res.Preview.Errors)
	}
	player := app.save.Slots[0].Player
	if player.Class != 1 || player.Dexterity != 16 || player.Intelligence != 10 || player.Arcane != 9 {
		t.Fatalf("class/minimum stats not persisted: class=%d dex=%d int=%d arc=%d",
			player.Class, player.Dexterity, player.Intelligence, player.Arcane)
	}
}

func TestApplyBuildTemplateV2_ClassMinimumRejectIsAtomic(t *testing.T) {
	app := makeDeriveFixture()
	before := snapPlayer(app.save.Slots[0].Player)
	beforeUndo := len(app.undoStacks[0])
	jsonText := makeV2Template(t, app, `{"stats":{"dexterity":true}}`, func(tpl *templates.BuildTemplate) {
		tpl.Sections.Stats.Dexterity = u32(13) // Warrior minimum is 16.
	})

	res, err := app.ApplyBuildTemplateV2ToCharacterJSON(0, jsonText, ApplyTemplateV2Options{
		DeriveLevelFromStats: true,
		ClassOverride:        &ClassOverride{ClassID: 1},
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if res.Applied || res.Preview.OK {
		t.Fatalf("expected rejected preview, got %+v", res)
	}
	if got := snapPlayer(app.save.Slots[0].Player); !reflect.DeepEqual(got, before) {
		t.Fatalf("player mutated on class-minimum reject:\n got %+v\nwant %+v", got, before)
	}
	if got := len(app.undoStacks[0]); got != beforeUndo {
		t.Fatalf("undo depth=%d, want unchanged %d", got, beforeUndo)
	}
}

func TestApplyBuildTemplateV2_UnknownClassRejectIsAtomic(t *testing.T) {
	app := makeDeriveFixture()
	before := snapPlayer(app.save.Slots[0].Player)
	beforeUndo := len(app.undoStacks[0])
	name := "Must Not Land"
	jsonText := makeV2Template(t, app, `{"profile":{"name":true}}`, func(tpl *templates.BuildTemplate) {
		tpl.Sections.Profile.Name = &name
	})

	res, err := app.ApplyBuildTemplateV2ToCharacterJSON(0, jsonText, ApplyTemplateV2Options{
		DeriveLevelFromStats: true,
		ClassOverride:        &ClassOverride{ClassID: 255},
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if res.Applied || res.Preview.OK {
		t.Fatalf("expected rejected preview, got %+v", res)
	}
	if len(res.Preview.Errors) == 0 || !strings.Contains(res.Preview.Errors[0].Message, "unknown class ID 255") {
		t.Fatalf("errors=%+v, want unknown class ID", res.Preview.Errors)
	}
	if got := snapPlayer(app.save.Slots[0].Player); !reflect.DeepEqual(got, before) {
		t.Fatalf("player mutated on unknown-class reject")
	}
	if got := len(app.undoStacks[0]); got != beforeUndo {
		t.Fatalf("undo depth=%d, want unchanged %d", got, beforeUndo)
	}
}

func TestApplyBuildTemplateV2_ClassOverrideRequiresDerivedLevel(t *testing.T) {
	app := makeDeriveFixture()
	before := snapPlayer(app.save.Slots[0].Player)
	name := "Must Not Land"
	jsonText := makeV2Template(t, app, `{"profile":{"name":true}}`, func(tpl *templates.BuildTemplate) {
		tpl.Sections.Profile.Name = &name
	})

	res, err := app.ApplyBuildTemplateV2ToCharacterJSON(0, jsonText, ApplyTemplateV2Options{
		ClassOverride: &ClassOverride{ClassID: 0},
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if res.Applied || res.Preview.OK {
		t.Fatalf("expected class/derive contract rejection, got %+v", res)
	}
	if got := snapPlayer(app.save.Slots[0].Player); !reflect.DeepEqual(got, before) {
		t.Fatalf("player mutated on class/derive contract reject")
	}
	if got := len(app.undoStacks[0]); got != 0 {
		t.Fatalf("undo depth=%d, want 0", got)
	}
}

func TestApplyBuildTemplateV2_ExplicitSoulMemoryBelowDerivedMinimumRejects(t *testing.T) {
	app := makeDeriveFixture()
	before := snapPlayer(app.save.Slots[0].Player)
	jsonText := makeV2Template(t, app, `{"profile":{"soulMemory":true},"stats":{"vigor":true}}`, func(tpl *templates.BuildTemplate) {
		tpl.Sections.Profile.SoulMemory = u32(1)
		tpl.Sections.Stats.Vigor = u32(60)
	})

	res, err := app.ApplyBuildTemplateV2ToCharacterJSON(0, jsonText, ApplyTemplateV2Options{
		DeriveLevelFromStats: true,
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if res.Applied || res.Preview.OK {
		t.Fatalf("expected Soul Memory rejection, got %+v", res)
	}
	if len(res.Preview.Errors) == 0 || !strings.Contains(strings.ToLower(res.Preview.Errors[0].Message), "soul memory") {
		t.Fatalf("errors=%+v, want Soul Memory error", res.Preview.Errors)
	}
	if got := snapPlayer(app.save.Slots[0].Player); !reflect.DeepEqual(got, before) {
		t.Fatalf("player mutated on Soul Memory reject")
	}
	if got := len(app.undoStacks[0]); got != 0 {
		t.Fatalf("undo depth=%d, want 0", got)
	}
}

func TestApplyBuildTemplateV2_ExplicitSoulMemoryAtMinimumSucceeds(t *testing.T) {
	app := makeDeriveFixture()
	// Effective final level is 68:
	// 60+15+18+16+13+9+9+7 - 79.
	const wantLevel = uint32(68)
	minimum := vm.MinimumSoulMemoryForLevel(wantLevel)
	jsonText := makeV2Template(t, app, `{"profile":{"soulMemory":true},"stats":{"vigor":true}}`, func(tpl *templates.BuildTemplate) {
		tpl.Sections.Profile.SoulMemory = u32(minimum)
		tpl.Sections.Stats.Vigor = u32(60)
	})

	res, err := app.ApplyBuildTemplateV2ToCharacterJSON(0, jsonText, ApplyTemplateV2Options{
		DeriveLevelFromStats: true,
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !res.Applied {
		t.Fatalf("Applied=false: %+v", res.Preview.Errors)
	}
	player := app.save.Slots[0].Player
	if player.Level != wantLevel || player.SoulMemory != minimum {
		t.Fatalf("level/soulMemory=(%d,%d), want (%d,%d)",
			player.Level, player.SoulMemory, wantLevel, minimum)
	}
}
