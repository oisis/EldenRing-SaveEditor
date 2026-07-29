package application

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/oisis/EldenRing-SaveForge/backend/core"
	"github.com/oisis/EldenRing-SaveForge/backend/editor"
	"github.com/oisis/EldenRing-SaveForge/backend/templates"
	"github.com/oisis/EldenRing-SaveForge/backend/vm"
)

// BuildTemplateV2ExportOptions is the Wails-facing input for v2 (profile +
// stats) template export from a character slot. Selection is passed
// separately as a JSON string so the boolean-or-map shape of
// templates.SectionSelection survives unchanged through Wails bindings.
//
// Unlike BuildTemplateExportOptions (v1, inventory.workspace), there is no
// IncludeInventory / IncludeStorage — section inclusion is driven entirely
// by the selection JSON.
type BuildTemplateV2ExportOptions struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Author      string   `json:"author"`
	Tags        []string `json:"tags"`
}

func u32p(v uint32) *uint32 { return &v }
func u8p(v uint8) *uint8    { return &v }

// parseTemplateSelectionJSON decodes the wire selection payload into a
// typed TemplateSelection. Top-level field names are restricted by
// DisallowUnknownFields so a typo like "profiel" surfaces as a hard error
// before BuildV2Template is reached. Per-section keys (level, vigor, ...)
// land in the open SectionSelection.Fields map and are validated downstream
// by templates.ValidateBuildTemplate's per-section allowlist.
func parseTemplateSelectionJSON(selectionJSON string) (*templates.TemplateSelection, error) {
	if strings.TrimSpace(selectionJSON) == "" {
		return nil, fmt.Errorf("selection JSON is empty")
	}
	dec := json.NewDecoder(strings.NewReader(selectionJSON))
	dec.DisallowUnknownFields()
	var sel templates.TemplateSelection
	if err := dec.Decode(&sel); err != nil {
		return nil, fmt.Errorf("parse selection: %w", err)
	}
	return &sel, nil
}

// buildTemplateV2SourcesFromCharacter maps a CharacterViewModel into the
// neutral DTOs consumed by templates.BuildV2Template. Mapping is dumb:
// every available field is copied for sections that have any selection;
// the builder then drops fields the user did not select (per-field
// selection) or keeps the whole section verbatim (boolean shortcut).
// Values are NOT clamped — out-of-range numbers surface as a builder /
// validator error rather than being silently coerced.
//
// vm.Souls maps to ProfileSource.Runes — the VM still carries the legacy
// "souls" naming from Dark Souls; the public template schema renamed the
// field to the Elden Ring term ("runes").
//
// ClassName is dropped when it begins with "Unknown (" — that prefix is
// the fallback emitted by vm.MapParsedSlotToVM when db.GetClassStats
// returns nil. A literal "Unknown (5)" string would round-trip through
// validation but is meaningless in a shareable template.
func buildTemplateV2SourcesFromCharacter(charVM *vm.CharacterViewModel, selection *templates.TemplateSelection) (*templates.ProfileSource, *templates.StatsSource) {
	var profile *templates.ProfileSource
	if selection.Profile.HasAny() {
		profile = &templates.ProfileSource{
			Name:                charVM.Name,
			Level:               u32p(charVM.Level),
			Runes:               u32p(charVM.Souls),
			SoulMemory:          u32p(charVM.SoulMemory),
			ClearCount:          u32p(charVM.ClearCount),
			ScadutreeBlessing:   u8p(charVM.ScadutreeBlessing),
			ShadowRealmBlessing: u8p(charVM.ShadowRealmBlessing),
			TalismanSlots:       u8p(charVM.TalismanSlots),
		}
		if !strings.HasPrefix(charVM.ClassName, "Unknown (") {
			profile.ClassName = charVM.ClassName
		}
	}
	var stats *templates.StatsSource
	if selection.Stats.HasAny() {
		stats = &templates.StatsSource{
			Vigor:        u32p(charVM.Vigor),
			Mind:         u32p(charVM.Mind),
			Endurance:    u32p(charVM.Endurance),
			Strength:     u32p(charVM.Strength),
			Dexterity:    u32p(charVM.Dexterity),
			Intelligence: u32p(charVM.Intelligence),
			Faith:        u32p(charVM.Faith),
			Arcane:       u32p(charVM.Arcane),
		}
	}
	return profile, stats
}

// buildAndValidateTemplateV2FromCharacter is the shared core for the v2
// charIndex-source endpoints. Phase 3C.1 ships only JSON export and
// preview; later phases (3C.2) reuse this helper for YAML export and
// library save without re-deriving sources.
//
// Every selected section of one export is derived from a single consistent
// slot snapshot: cloneCharacterSlot takes one deep copy under one
// saveMu.RLock + one slotMu[charIndex] hold, then releases both. All section
// builders below run on that detached copy with no locks held, so a
// concurrent writer mutating the live slot mid-build can never produce a
// "torn template" whose sections come from different slot states.
func (a *App) buildAndValidateTemplateV2FromCharacter(charIndex int, selectionJSON string, opts BuildTemplateV2ExportOptions) (*templates.BuildTemplate, string, error) {
	selection, err := parseTemplateSelectionJSON(selectionJSON)
	if err != nil {
		return nil, "", err
	}
	slot, err := a.cloneCharacterSlot(charIndex)
	if err != nil {
		return nil, "", err
	}
	charVM, err := vm.MapParsedSlotToVM(slot)
	if err != nil {
		return nil, "", err
	}
	profile, stats := buildTemplateV2SourcesFromCharacter(charVM, selection)
	var itemsSource *templates.ItemsLayoutSource
	if selection.Items.HasAny() ||
		selection.InventoryLayout.HasAny() ||
		selection.StorageLayout.HasAny() {
		itemsSource, err = buildItemsSourceFromSlot(slot, charIndex)
		if err != nil {
			return nil, "", fmt.Errorf("build v2 items source: %w", err)
		}
	}
	var equipment *templates.EquipmentSection
	var equippedSpellsRaw []uint32
	var equippedSpellsLimit int
	if selection.Equipment.HasAny() || selection.Spells.HasAny() {
		equipment, equippedSpellsRaw, equippedSpellsLimit, err = buildEquipmentSpellsSourcesFromSlot(
			slot, charIndex, selection.Equipment.HasAny(), selection.Spells.HasAny())
		if err != nil {
			return nil, "", fmt.Errorf("build v2 equipment/spells source: %w", err)
		}
	}
	tags := opts.Tags
	if tags == nil {
		tags = []string{}
	}
	tpl, err := templates.BuildV2Template(templates.ExportV2Options{
		AppVersion: appVersion,
		Metadata: &templates.TemplateMetadata{
			Name:                 opts.Name,
			Description:          opts.Description,
			Author:               opts.Author,
			Tags:                 tags,
			SourceCharacterIndex: charIndex,
			SourceCharacterName:  charVM.Name,
		},
		Profile:             profile,
		Stats:               stats,
		ItemsSource:         itemsSource,
		Equipment:           equipment,
		EquippedSpellsRaw:   equippedSpellsRaw,
		EquippedSpellsLimit: equippedSpellsLimit,
		Selection:           selection,
	})
	if err != nil {
		return nil, "", fmt.Errorf("build v2 template: %w", err)
	}
	data, err := marshalBuildTemplate(tpl)
	if err != nil {
		return nil, "", err
	}
	return tpl, string(data), nil
}

// ExportBuildTemplateV2JSONFromCharacter returns the canonical JSON for a
// v2 template built from slot charIndex. Selection is a JSON-encoded
// templates.TemplateSelection — see parseTemplateSelectionJSON for the
// accepted shape.
//
// Dialog-less and filesystem-less. Use it from a UI that has its own
// preview / save UX (Phase 3D) or from tests.
func (a *App) ExportBuildTemplateV2JSONFromCharacter(charIndex int, selectionJSON string, opts BuildTemplateV2ExportOptions) (string, error) {
	_, jsonText, err := a.buildAndValidateTemplateV2FromCharacter(charIndex, selectionJSON, opts)
	return jsonText, err
}

// PreviewBuildTemplateV2FromCharacter builds a v2 template from slot
// charIndex and runs the preview validator. The returned LoadedTemplatePreview
// carries the canonical JSON alongside the report so a follow-up "save to
// library" call (Phase 3C.2) can reuse the exact same bytes — the
// anti-TOCTOU pattern already used by the YAML import path.
func (a *App) PreviewBuildTemplateV2FromCharacter(charIndex int, selectionJSON string, opts BuildTemplateV2ExportOptions) (LoadedTemplatePreview, error) {
	tpl, jsonText, err := a.buildAndValidateTemplateV2FromCharacter(charIndex, selectionJSON, opts)
	if err != nil {
		return LoadedTemplatePreview{}, err
	}
	report := templates.PreviewBuildTemplateImport(tpl, templates.ImportPreviewOptions{Mode: "append"})
	return LoadedTemplatePreview{Report: report, JSON: jsonText}, nil
}

// ExportBuildTemplateV2YAMLFromCharacter returns the canonical YAML payload
// for a v2 template built from slot charIndex. Dialog-less and
// filesystem-less — the file save dialog ships in Phase 3D alongside the
// bindings regen and UI.
//
// The output round-trips through templates.ParseBuildTemplateYAML: schema,
// version: 2, selection (boolean shortcut or per-field map per the
// requested selection), and sections.profile / sections.stats survive
// untouched.
func (a *App) ExportBuildTemplateV2YAMLFromCharacter(charIndex int, selectionJSON string, opts BuildTemplateV2ExportOptions) (string, error) {
	tpl, _, err := a.buildAndValidateTemplateV2FromCharacter(charIndex, selectionJSON, opts)
	if err != nil {
		return "", err
	}
	data, err := templates.MarshalBuildTemplateYAML(tpl)
	if err != nil {
		return "", fmt.Errorf("marshal v2 yaml: %w", err)
	}
	return string(data), nil
}

// SaveBuildTemplateV2FromCharacterToLibrary builds a v2 template from slot
// charIndex and persists it in the local library. Returns the new index
// entry (Version=2, SelectedSections populated by Phase 3C.0). Library
// re-validates and re-marshals the template internally; the canonical JSON
// from buildAndValidateTemplateV2FromCharacter is intentionally discarded
// to keep on-disk encoding centralised in Library.SaveTemplate.
//
// opts.Name may be empty — Library falls back to a "template-" filename
// stem when no display name is provided.
func (a *App) SaveBuildTemplateV2FromCharacterToLibrary(charIndex int, selectionJSON string, opts BuildTemplateV2ExportOptions) (templates.LibraryTemplateEntry, error) {
	tpl, _, err := a.buildAndValidateTemplateV2FromCharacter(charIndex, selectionJSON, opts)
	if err != nil {
		return templates.LibraryTemplateEntry{}, err
	}
	lib, err := a.ensureTemplateLibrary()
	if err != nil {
		return templates.LibraryTemplateEntry{}, err
	}
	return lib.SaveTemplate(tpl)
}

// cloneCharacterSlot returns a deep, independent copy of slot charIndex taken
// under one saveMu.RLock + one slotMu[charIndex] hold. The copy is detached
// from a.save, so every export section can be built from it after the locks
// are released without risking a torn read against a concurrent writer.
//
// It replaces the previous per-section lock acquisitions (GetCharacter +
// buildItemsSourceForCharacter + buildEquipmentSpellsSourcesForCharacter),
// which each grabbed the same locks separately and could therefore mix data
// from different slot states into one template.
func (a *App) cloneCharacterSlot(charIndex int) (*core.SaveSlot, error) {
	a.saveMu.RLock()
	defer a.saveMu.RUnlock()
	if a.save == nil {
		return nil, fmt.Errorf("no save loaded")
	}
	if charIndex < 0 || charIndex >= len(a.save.Slots) {
		return nil, fmt.Errorf("invalid slot index %d", charIndex)
	}
	a.slotMu[charIndex].Lock()
	defer a.slotMu[charIndex].Unlock()
	return core.CloneSlot(&a.save.Slots[charIndex]), nil
}

// buildItemsSourceFromSlot builds a read-only ItemsLayoutSource from an
// already-detached slot snapshot. Lock-free by contract: the caller
// (cloneCharacterSlot) has already isolated the slot bytes.
func buildItemsSourceFromSlot(slot *core.SaveSlot, charIndex int) (*templates.ItemsLayoutSource, error) {
	snap, err := editor.BuildSnapshot(slot, "", charIndex)
	if err != nil {
		return nil, err
	}
	return &templates.ItemsLayoutSource{
		InventoryItems: snap.InventoryItems,
		StorageItems:   snap.StorageItems,
	}, nil
}

// buildEquipmentSpellsSourcesFromSlot reads the equipped loadout and the raw
// 14-slot spell loadout from an already-detached slot snapshot. Lock-free by
// contract — see cloneCharacterSlot. Read-only: the snapshot is never mutated.
//
// Both sections come from ONE core.SaveSlot.ReadEquippedState call, so
// equipment and spells are always drawn from the same equipped-armaments read
// (single dynamic projectile-count decode, no second reader):
//   - equipment: RawEquippedState.Equipped is the real equipped-armaments
//     block (the ChrAsm2 GaItem-handle header at EquipItemsIDOffset is a decoy
//     the item DB cannot resolve and is never read here).
//     buildEquipmentSectionFromEquipped runs with emitEmptyAsClear=true, so an
//     empty writable slot exports as an explicit clear (BaseItemID == 0) and
//     applying the template strips stale gear from the target slot;
//   - spells: RawEquippedState.Spells carries the save's empty-slot sentinel
//     0xFFFFFFFF for unused slots, which BuildV2Template maps to the same
//     explicit-clear form (BaseItemID == 0).
//
// Talisman slots are exported only up to the source character's active pouch
// capacity; slots still locked beyond that active capacity stay nil rather
// than emitting clears (the source never had that pouch slot to clear). A
// slot whose equipped state cannot be read yields a hard error so the caller
// fails closed rather than emitting a partial "full loadout".
func buildEquipmentSpellsSourcesFromSlot(slot *core.SaveSlot, charIndex int, needEquipment, needSpells bool) (*templates.EquipmentSection, []uint32, int, error) {
	raw, err := slot.ReadEquippedState()
	if err != nil {
		return nil, nil, 0, fmt.Errorf("read equipped spells/equipment state: %w", err)
	}

	var equipment *templates.EquipmentSection
	if needEquipment {
		snap, err := editor.BuildSnapshot(slot, "", charIndex)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("build equipment snapshot: %w", err)
		}
		activeTalismans := activeTalismanSlotCount(slot.Player.TalismanSlots)
		equipment = buildEquipmentSectionFromEquipped(raw.Equipped, snap.InventoryItems, activeTalismans, true)
		if equipment == nil {
			return nil, nil, 0, fmt.Errorf("equipment loadout unavailable for slot %d", charIndex)
		}
	}

	var equippedSpellsRaw []uint32
	var equippedSpellsLimit int
	if needSpells {
		equippedSpellsRaw = append([]uint32(nil), raw.Spells[:]...)
		equippedSpellsLimit = templateSpellSlotLimit(slot, raw)
	}

	return equipment, equippedSpellsRaw, equippedSpellsLimit, nil
}
