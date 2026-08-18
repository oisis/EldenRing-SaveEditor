/*
Endpoint: GetBuildTemplatePreview
EndpointID: get_build_template_preview
Purpose: Builds a non-mutating preview of applying a template to the specified character.
How it works: The runtime handler resolves templateID through the templates store, reads the character profile, statistics, and equipped spells from the private save session snapshot, builds a deterministic diff plan against the template's target values under save revision consistency checks, and reports whether the plan is executable without mutating save or library state.
Supported resource types: GameResource references.
Input variables: saveSessionID, characterID, templateID, selection, options.
GameCatalog variables read: for occupied spell memory slots (1-12), the presentation name and memory slot cost from GameCatalog.
Save variables read: the UserData10 activity flag and profile of the target character, starting class, stats attributes, and equipped spells; the getter is strictly non-mutating.
Implementation status: implemented
*/
package templates

import (
	"errors"
	"fmt"
	"strings"

	"github.com/oisis/EldenRing-SaveForge/backend/buildtemplates"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/equipment"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// GetBuildTemplatePreviewEndpointID is the stable backend identifier of GetBuildTemplatePreview.
const GetBuildTemplatePreviewEndpointID = "get_build_template_preview"

// Stable issue codes reported in blocking issues.
const (
	IssueCodeEmptySelection         = "empty_selection"
	IssueCodeUnsupportedSection     = "unsupported_section"
	IssueCodeUnsupportedField       = "unsupported_field"
	IssueCodeSelectionNotInTemplate = "selection_not_in_template"
	IssueCodeMissingSection         = "missing_section"
	IssueCodeMissingValue           = "missing_value"
	IssueCodeUnsupportedOption      = "unsupported_option"
	IssueCodeInvalidStats           = "invalid_stats"
	IssueCodeLevelMismatch          = "level_mismatch"
	IssueCodeInvalidSpellLoadout    = "invalid_spell_loadout"
)

// GetBuildTemplatePreviewDefinition describes the public getter contract.
var GetBuildTemplatePreviewDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetBuildTemplatePreview",
	ID:                         GetBuildTemplatePreviewEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "GameResource references",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "templateID", "selection", "options"},
	Description:                "Builds a non-mutating preview of applying a template to the specified character.",
})

// GetBuildTemplatePreviewRequest is the typed request for GetBuildTemplatePreview.
type GetBuildTemplatePreviewRequest struct {
	SaveSessionID string                            `json:"saveSessionID"`
	CharacterID   int                               `json:"characterID"`
	TemplateID    string                            `json:"templateID"`
	Selection     *buildtemplates.TemplateSelection `json:"selection,omitempty"`
	Options       *buildtemplates.ApplyOptions      `json:"options,omitempty"`
}

// GetBuildTemplatePreviewResult is the typed return of GetBuildTemplatePreview.
type GetBuildTemplatePreviewResult struct {
	TemplateID       string                      `json:"templateID"`
	TemplateRevision string                      `json:"templateRevision"`
	CharacterID      int                         `json:"characterID"`
	SaveSessionID    string                      `json:"saveSessionID"`
	SaveRevision     string                      `json:"saveRevision"`
	Executable       bool                        `json:"executable"`
	Plan             BuildTemplatePreviewPlan    `json:"plan"`
	BlockingIssues   []BuildTemplatePreviewIssue `json:"blockingIssues,omitempty"`
}

// BuildTemplatePreviewIssue describes an issue or limitation encountered during planning.
type BuildTemplatePreviewIssue struct {
	Code    string `json:"code"`
	Section string `json:"section,omitempty"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

// BuildTemplatePreviewPlan holds the comparison plan for each selected section.
type BuildTemplatePreviewPlan struct {
	Profile *ProfilePreviewPlan `json:"profile,omitempty"`
	Stats   *StatsPreviewPlan   `json:"stats,omitempty"`
	Spells  *SpellsPreviewPlan  `json:"spells,omitempty"`
}

// ProfilePreviewPlan describes the planned profile field changes.
type ProfilePreviewPlan struct {
	Name  *StringFieldChange `json:"name,omitempty"`
	Level *Uint32FieldChange `json:"level,omitempty"`
}

// StatsPreviewPlan describes the planned attribute changes and resulting derived values.
type StatsPreviewPlan struct {
	Vigor            *Uint32FieldChange `json:"vigor,omitempty"`
	Mind             *Uint32FieldChange `json:"mind,omitempty"`
	Endurance        *Uint32FieldChange `json:"endurance,omitempty"`
	Strength         *Uint32FieldChange `json:"strength,omitempty"`
	Dexterity        *Uint32FieldChange `json:"dexterity,omitempty"`
	Intelligence     *Uint32FieldChange `json:"intelligence,omitempty"`
	Faith            *Uint32FieldChange `json:"faith,omitempty"`
	Arcane           *Uint32FieldChange `json:"arcane,omitempty"`
	ResultLevel      uint32             `json:"resultLevel"`
	ResultSoulMemory uint32             `json:"resultSoulMemory"`
}

// SpellsPreviewPlan describes the planned spell memory changes and final compact loadout.
type SpellsPreviewPlan struct {
	Slots                []SpellSlotChange             `json:"slots,omitempty"`
	UsedMemorySlots      int                           `json:"usedMemorySlots"`
	AvailableMemorySlots int                           `json:"availableMemorySlots"`
	EquippedSpells       []buildtemplates.SpellSlotRef `json:"equippedSpells,omitempty"`
}

// StringFieldChange represents a string comparison between current and target state.
type StringFieldChange struct {
	Current string `json:"current"`
	Target  string `json:"target"`
	Changed bool   `json:"changed"`
}

// Uint32FieldChange represents a numeric comparison between current and target state.
type Uint32FieldChange struct {
	Current uint32 `json:"current"`
	Target  uint32 `json:"target"`
	Changed bool   `json:"changed"`
}

// SpellSlotChange represents a comparison for one of the 12 spell memory slots.
type SpellSlotChange struct {
	SlotNumber int                          `json:"slotNumber"`
	Current    *buildtemplates.SpellSlotRef `json:"current,omitempty"`
	Target     *buildtemplates.SpellSlotRef `json:"target,omitempty"`
	Changed    bool                         `json:"changed"`
}

// GetBuildTemplatePreview builds a non-mutating preview plan for applying a template.
func GetBuildTemplatePreview(
	store *buildtemplates.Store,
	engine *saveengine.Engine,
	catalog *gamecatalog.Catalog,
	req GetBuildTemplatePreviewRequest,
) (GetBuildTemplatePreviewResult, error) {
	resolved, err := planBuildTemplate(
		store,
		engine,
		catalog,
		req.SaveSessionID,
		req.CharacterID,
		req.TemplateID,
		req.Selection,
		req.Options,
	)
	if err != nil {
		return GetBuildTemplatePreviewResult{}, err
	}
	return resolved.previewResult, nil
}

type resolvedBuildTemplatePlan struct {
	previewResult GetBuildTemplatePreviewResult
	targetName    *string
	targetAttrs   *saveengine.CharacterAttributes
	targetSpells  *saveengine.CharacterSpellsPlan
}

// planBuildTemplate builds a preview plan and resolves the validated target mutation state.
// It is the single shared planner used by GetBuildTemplatePreview and ApplyBuildTemplate.
func planBuildTemplate(
	store *buildtemplates.Store,
	engine *saveengine.Engine,
	catalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	templateID string,
	selection *buildtemplates.TemplateSelection,
	options *buildtemplates.ApplyOptions,
) (resolvedBuildTemplatePlan, error) {
	if store == nil {
		return resolvedBuildTemplatePlan{}, errors.New("templates store is not available")
	}
	if engine == nil {
		return resolvedBuildTemplatePlan{}, errors.New("save engine is not available")
	}
	if catalog == nil {
		return resolvedBuildTemplatePlan{}, errors.New("game catalog is not available")
	}
	if saveSessionID == "" {
		return resolvedBuildTemplatePlan{}, errors.New("saveSessionID must not be empty")
	}
	if templateID == "" {
		return resolvedBuildTemplatePlan{}, errors.New("templateID must not be empty")
	}
	if characterID < 0 || characterID > 9 {
		return resolvedBuildTemplatePlan{}, fmt.Errorf("characterID %d out of range (0..9)", characterID)
	}

	// 1. Load template from store.
	tpl, templateRevision, err := store.GetTemplate(templateID)
	if err != nil {
		return resolvedBuildTemplatePlan{}, err
	}

	// 2. Initial save read consistency check.
	initialUndo, err := engine.GetUndoState(saveSessionID, characterID)
	if err != nil {
		return resolvedBuildTemplatePlan{}, err
	}

	// 3. Verify character slot is active.
	profile, err := engine.GetCharacterProfile(saveSessionID, characterID)
	if err != nil {
		return resolvedBuildTemplatePlan{}, err
	}
	if !profile.Active {
		return resolvedBuildTemplatePlan{}, fmt.Errorf("character slot %d is inactive", characterID)
	}

	var blockingIssues []BuildTemplatePreviewIssue
	var plan BuildTemplatePreviewPlan

	var targetName *string
	var targetAttrs *saveengine.CharacterAttributes
	var targetSpells *saveengine.CharacterSpellsPlan

	// 4. Validate and resolve selection narrowing.
	baseSelection := tpl.Selection
	if baseSelection == nil {
		// v1 template fallback base selection.
		baseSelection = &buildtemplates.TemplateSelection{
			InventoryWorkspace: &buildtemplates.SectionSelection{All: true},
		}
	}

	var effectiveSelection *buildtemplates.TemplateSelection
	if selection == nil {
		effectiveSelection = baseSelection
	} else {
		if err := buildtemplates.ValidateTemplateSelection(selection); err != nil {
			blockingIssues = append(blockingIssues, BuildTemplatePreviewIssue{
				Code:    IssueCodeUnsupportedField,
				Message: fmt.Sprintf("invalid selection: %v", err),
			})
		}
		if !selection.HasAnySelected() {
			blockingIssues = append(blockingIssues, BuildTemplatePreviewIssue{
				Code:    IssueCodeEmptySelection,
				Message: "selection contains no selected sections or fields",
			})
		} else {
			// Enforce narrowing rule: selection can only disable or pick a subset of baseSelection.
			checkSubset := func(secName string, reqSec, baseSec *buildtemplates.SectionSelection) {
				if reqSec == nil || !reqSec.HasAny() {
					return
				}
				if baseSec == nil || !baseSec.HasAny() {
					blockingIssues = append(blockingIssues, BuildTemplatePreviewIssue{
						Code:    IssueCodeSelectionNotInTemplate,
						Section: secName,
						Message: fmt.Sprintf("section %q is not selected in the template", secName),
					})
					return
				}
				if reqSec.All {
					if !baseSec.All {
						blockingIssues = append(blockingIssues, BuildTemplatePreviewIssue{
							Code:    IssueCodeSelectionNotInTemplate,
							Section: secName,
							Message: fmt.Sprintf("section %q has All: true in selection, but template does not select all fields", secName),
						})
					}
					return
				}
				for field, sel := range reqSec.Fields {
					if !sel {
						continue
					}
					if !baseSec.All && (baseSec.Fields == nil || !baseSec.Fields[field]) {
						blockingIssues = append(blockingIssues, BuildTemplatePreviewIssue{
							Code:    IssueCodeSelectionNotInTemplate,
							Section: secName,
							Field:   field,
							Message: fmt.Sprintf("field %q in section %q is not selected in the template", field, secName),
						})
					}
				}
			}

			checkSubset("inventory.workspace", selection.InventoryWorkspace, baseSelection.InventoryWorkspace)
			checkSubset("profile", selection.Profile, baseSelection.Profile)
			checkSubset("stats", selection.Stats, baseSelection.Stats)
			checkSubset("equipment", selection.Equipment, baseSelection.Equipment)
			checkSubset("spells", selection.Spells, baseSelection.Spells)
			checkSubset("items", selection.Items, baseSelection.Items)
			checkSubset("inventoryLayout", selection.InventoryLayout, baseSelection.InventoryLayout)
			checkSubset("storageLayout", selection.StorageLayout, baseSelection.StorageLayout)
		}
		effectiveSelection = selection
	}

	// 5. Validate options: unsupported options in tpl.ApplyOptions or options create blocking issues.
	checkApplyOptions := func(source string, opts *buildtemplates.ApplyOptions) {
		if opts == nil {
			return
		}
		if opts.Items != nil {
			blockingIssues = append(blockingIssues, BuildTemplatePreviewIssue{
				Code:    IssueCodeUnsupportedOption,
				Field:   "items",
				Message: fmt.Sprintf("%s.items is not supported for template application in this version", source),
			})
		}
		if opts.InventoryLayout != nil {
			blockingIssues = append(blockingIssues, BuildTemplatePreviewIssue{
				Code:    IssueCodeUnsupportedOption,
				Field:   "inventoryLayout",
				Message: fmt.Sprintf("%s.inventoryLayout is not supported for template application in this version", source),
			})
		}
		if opts.StorageLayout != nil {
			blockingIssues = append(blockingIssues, BuildTemplatePreviewIssue{
				Code:    IssueCodeUnsupportedOption,
				Field:   "storageLayout",
				Message: fmt.Sprintf("%s.storageLayout is not supported for template application in this version", source),
			})
		}
		if opts.WeaponLevelOverride != nil {
			blockingIssues = append(blockingIssues, BuildTemplatePreviewIssue{
				Code:    IssueCodeUnsupportedOption,
				Field:   "weaponLevelOverride",
				Message: fmt.Sprintf("%s.weaponLevelOverride is not supported for template application in this version", source),
			})
		}
	}
	checkApplyOptions("template.applyOptions", tpl.ApplyOptions)
	checkApplyOptions("applyOptions", options)

	// 6. Check unsupported sections in effective selection.
	if effectiveSelection.InventoryWorkspace.HasAny() {
		blockingIssues = append(blockingIssues, BuildTemplatePreviewIssue{
			Code:    IssueCodeUnsupportedSection,
			Section: "inventory.workspace",
			Message: "inventory.workspace section is not supported for template application in this version",
		})
	}
	if effectiveSelection.Equipment.HasAny() {
		blockingIssues = append(blockingIssues, BuildTemplatePreviewIssue{
			Code:    IssueCodeUnsupportedSection,
			Section: "equipment",
			Message: "equipment section is not supported for template application in this version",
		})
	}
	if effectiveSelection.Items.HasAny() {
		blockingIssues = append(blockingIssues, BuildTemplatePreviewIssue{
			Code:    IssueCodeUnsupportedSection,
			Section: "items",
			Message: "items section is not supported for template application in this version",
		})
	}
	if effectiveSelection.InventoryLayout.HasAny() {
		blockingIssues = append(blockingIssues, BuildTemplatePreviewIssue{
			Code:    IssueCodeUnsupportedSection,
			Section: "inventoryLayout",
			Message: "inventoryLayout section is not supported for template application in this version",
		})
	}
	if effectiveSelection.StorageLayout.HasAny() {
		blockingIssues = append(blockingIssues, BuildTemplatePreviewIssue{
			Code:    IssueCodeUnsupportedSection,
			Section: "storageLayout",
			Message: "storageLayout section is not supported for template application in this version",
		})
	}

	// 7. Stats section planning.
	var calculatedStatsLevel *uint32
	if effectiveSelection.Stats.HasAny() {
		if tpl.Sections.Stats == nil {
			blockingIssues = append(blockingIssues, BuildTemplatePreviewIssue{
				Code:    IssueCodeMissingSection,
				Section: "stats",
				Message: "template does not contain a stats section",
			})
		} else {
			curStats, err := engine.GetCharacterStats(saveSessionID, characterID)
			if err != nil {
				return resolvedBuildTemplatePlan{}, err
			}
			sPlan := &StatsPreviewPlan{}
			selAll := effectiveSelection.Stats.All
			fields := effectiveSelection.Stats.Fields

			resolvedAttrs := saveengine.CharacterAttributes{
				Vigor:        curStats.Vigor,
				Mind:         curStats.Mind,
				Endurance:    curStats.Endurance,
				Strength:     curStats.Strength,
				Dexterity:    curStats.Dexterity,
				Intelligence: curStats.Intelligence,
				Faith:        curStats.Faith,
				Arcane:       curStats.Arcane,
			}

			checkStat := func(field string, curVal uint32, tgtValPtr *uint32, targetAttr *uint32) *Uint32FieldChange {
				if !selAll && !fields[field] {
					return nil
				}
				if tgtValPtr == nil {
					blockingIssues = append(blockingIssues, BuildTemplatePreviewIssue{
						Code:    IssueCodeMissingValue,
						Section: "stats",
						Field:   field,
						Message: fmt.Sprintf("template stats section does not contain a %s value", field),
					})
					return nil
				}
				tgtVal := *tgtValPtr
				*targetAttr = tgtVal
				return &Uint32FieldChange{
					Current: curVal,
					Target:  tgtVal,
					Changed: curVal != tgtVal,
				}
			}

			sPlan.Vigor = checkStat("vigor", curStats.Vigor, tpl.Sections.Stats.Vigor, &resolvedAttrs.Vigor)
			sPlan.Mind = checkStat("mind", curStats.Mind, tpl.Sections.Stats.Mind, &resolvedAttrs.Mind)
			sPlan.Endurance = checkStat("endurance", curStats.Endurance, tpl.Sections.Stats.Endurance, &resolvedAttrs.Endurance)
			sPlan.Strength = checkStat("strength", curStats.Strength, tpl.Sections.Stats.Strength, &resolvedAttrs.Strength)
			sPlan.Dexterity = checkStat("dexterity", curStats.Dexterity, tpl.Sections.Stats.Dexterity, &resolvedAttrs.Dexterity)
			sPlan.Intelligence = checkStat("intelligence", curStats.Intelligence, tpl.Sections.Stats.Intelligence, &resolvedAttrs.Intelligence)
			sPlan.Faith = checkStat("faith", curStats.Faith, tpl.Sections.Stats.Faith, &resolvedAttrs.Faith)
			sPlan.Arcane = checkStat("arcane", curStats.Arcane, tpl.Sections.Stats.Arcane, &resolvedAttrs.Arcane)

			plannedLevel, plannedSoulMemory, planErr := engine.PlanCharacterStats(saveSessionID, characterID, resolvedAttrs)
			if planErr != nil {
				blockingIssues = append(blockingIssues, BuildTemplatePreviewIssue{
					Code:    IssueCodeInvalidStats,
					Section: "stats",
					Message: planErr.Error(),
				})
			} else {
				sPlan.ResultLevel = plannedLevel
				sPlan.ResultSoulMemory = plannedSoulMemory
				calculatedStatsLevel = &plannedLevel
				targetAttrs = &resolvedAttrs
			}
			plan.Stats = sPlan
		}
	}

	// 8. Profile section planning.
	if effectiveSelection.Profile.HasAny() {
		if tpl.Sections.Profile == nil {
			blockingIssues = append(blockingIssues, BuildTemplatePreviewIssue{
				Code:    IssueCodeMissingSection,
				Section: "profile",
				Message: "template does not contain a profile section",
			})
		} else {
			if effectiveSelection.Profile.All {
				blockingIssues = append(blockingIssues, BuildTemplatePreviewIssue{
					Code:    IssueCodeUnsupportedField,
					Section: "profile",
					Message: "selection.profile boolean shortcut (All) is not supported because unconfirmed profile fields cannot be applied",
				})
			}
			for field, selected := range effectiveSelection.Profile.Fields {
				if !selected {
					continue
				}
				if field != "name" && field != "level" {
					blockingIssues = append(blockingIssues, BuildTemplatePreviewIssue{
						Code:    IssueCodeUnsupportedField,
						Section: "profile",
						Field:   field,
						Message: fmt.Sprintf("profile field %q is not supported for template application in this version", field),
					})
				}
			}
			pPlan := &ProfilePreviewPlan{}
			if effectiveSelection.Profile.Fields["name"] {
				if tpl.Sections.Profile.Name == nil {
					blockingIssues = append(blockingIssues, BuildTemplatePreviewIssue{
						Code:    IssueCodeMissingValue,
						Section: "profile",
						Field:   "name",
						Message: "template profile section does not contain a name value",
					})
				} else {
					curVal := profile.Name
					tgtVal := *tpl.Sections.Profile.Name
					pPlan.Name = &StringFieldChange{
						Current: curVal,
						Target:  tgtVal,
						Changed: curVal != tgtVal,
					}
					targetName = &tgtVal
				}
			}
			if effectiveSelection.Profile.Fields["level"] {
				if !effectiveSelection.Stats.HasAny() {
					blockingIssues = append(blockingIssues, BuildTemplatePreviewIssue{
						Code:    IssueCodeUnsupportedField,
						Section: "profile",
						Field:   "level",
						Message: "profile.level cannot be applied without stats",
					})
				} else if tpl.Sections.Profile.Level == nil {
					blockingIssues = append(blockingIssues, BuildTemplatePreviewIssue{
						Code:    IssueCodeMissingValue,
						Section: "profile",
						Field:   "level",
						Message: "template profile section does not contain a level value",
					})
				} else {
					curVal := profile.Level
					tgtVal := *tpl.Sections.Profile.Level
					pPlan.Level = &Uint32FieldChange{
						Current: curVal,
						Target:  tgtVal,
						Changed: curVal != tgtVal,
					}
					if calculatedStatsLevel != nil && tgtVal != *calculatedStatsLevel {
						blockingIssues = append(blockingIssues, BuildTemplatePreviewIssue{
							Code:    IssueCodeLevelMismatch,
							Section: "profile",
							Field:   "level",
							Message: fmt.Sprintf("template profile.level %d does not match calculated stats level %d", tgtVal, *calculatedStatsLevel),
						})
					}
				}
			}
			plan.Profile = pPlan
		}
	}

	// 9. Spells section planning.
	if effectiveSelection.Spells.HasAny() {
		if tpl.Sections.Spells == nil {
			blockingIssues = append(blockingIssues, BuildTemplatePreviewIssue{
				Code:    IssueCodeMissingSection,
				Section: "spells",
				Message: "template does not contain a spells section",
			})
		} else {
			if effectiveSelection.Spells.All {
				blockingIssues = append(blockingIssues, BuildTemplatePreviewIssue{
					Code:    IssueCodeUnsupportedField,
					Section: "spells",
					Message: "selection.spells boolean shortcut (All) is not supported because unconfirmed slots 13-14 cannot be applied",
				})
			}
			if effectiveSelection.Spells.Fields["spell13"] || effectiveSelection.Spells.Fields["spell14"] {
				blockingIssues = append(blockingIssues, BuildTemplatePreviewIssue{
					Code:    IssueCodeUnsupportedField,
					Section: "spells",
					Message: "spells slots 13 and 14 are not supported (public contract supports slots 1-12)",
				})
			}

			rawSpells, err := engine.GetEquippedSpells(saveSessionID, characterID)
			if err != nil {
				return resolvedBuildTemplatePlan{}, err
			}
			if rawSpells.Spells[12] != 0xFFFFFFFF || rawSpells.Spells[13] != 0xFFFFFFFF {
				blockingIssues = append(blockingIssues, BuildTemplatePreviewIssue{
					Code:    IssueCodeInvalidSpellLoadout,
					Section: "spells",
					Message: "character physical spell slots 13 and 14 are occupied; cannot apply spell loadout",
				})
			}

			spellsRes, err := equipment.GetEquippedSpells(engine, catalog, saveSessionID, characterID)
			if err != nil {
				return resolvedBuildTemplatePlan{}, err
			}

			// Step 9.1: Build 12-position intermediate array with current vs template targets.
			tplSpells := []*buildtemplates.SpellSlotRef{
				tpl.Sections.Spells.Spell1, tpl.Sections.Spells.Spell2, tpl.Sections.Spells.Spell3,
				tpl.Sections.Spells.Spell4, tpl.Sections.Spells.Spell5, tpl.Sections.Spells.Spell6,
				tpl.Sections.Spells.Spell7, tpl.Sections.Spells.Spell8, tpl.Sections.Spells.Spell9,
				tpl.Sections.Spells.Spell10, tpl.Sections.Spells.Spell11, tpl.Sections.Spells.Spell12,
			}

			var intermediateTargets [12]*buildtemplates.SpellSlotRef
			for i := 0; i < 12; i++ {
				key := fmt.Sprintf("spell%d", i+1)
				var curRef *buildtemplates.SpellSlotRef
				if i < len(spellsRes.Spells) {
					s := spellsRes.Spells[i]
					if s.RawMagicParamID != 0 && s.RawMagicParamID != 0xFFFFFFFF {
						curRef = &buildtemplates.SpellSlotRef{
							BaseItemID: buildtemplates.SpellItemIDPrefix | s.RawMagicParamID,
							Name:       s.Name,
						}
					}
				}
				if effectiveSelection.Spells.Fields[key] {
					intermediateTargets[i] = tplSpells[i]
				} else {
					intermediateTargets[i] = curRef
				}
			}

			// Step 9.2: Form compact sequence without gaps.
			var compactTargets []*buildtemplates.SpellSlotRef
			for _, tgt := range intermediateTargets {
				if tgt != nil {
					compactTargets = append(compactTargets, tgt)
				}
			}

			// Step 9.3: Validate compact sequence through GameCatalog.
			spellLoadoutValid := true
			seenRawIDs := make(map[uint32]struct{}, len(compactTargets))
			usedMemorySlots := 0
			var resolvedEquipped []buildtemplates.SpellSlotRef
			var resolvedRawIDs []uint32

			for idx, tgt := range compactTargets {
				res, exists := catalog.ItemByGameID(tgt.BaseItemID)
				if !exists {
					spellLoadoutValid = false
					blockingIssues = append(blockingIssues, BuildTemplatePreviewIssue{
						Code:    IssueCodeInvalidSpellLoadout,
						Section: "spells",
						Message: fmt.Sprintf("target spell[%d] 0x%08X is not found in game catalog", idx, tgt.BaseItemID),
					})
					continue
				}
				rawID, memCost, valErr := gamecatalog.ValidateSpellResource(res)
				if valErr != nil {
					spellLoadoutValid = false
					blockingIssues = append(blockingIssues, BuildTemplatePreviewIssue{
						Code:    IssueCodeInvalidSpellLoadout,
						Section: "spells",
						Message: fmt.Sprintf("target spell[%d]: %v", idx, valErr),
					})
					continue
				}
				if res.Item == nil || !res.Item.Presentation.Name.Known || res.Item.Presentation.Name.Value == "" {
					spellLoadoutValid = false
					blockingIssues = append(blockingIssues, BuildTemplatePreviewIssue{
						Code:    IssueCodeInvalidSpellLoadout,
						Section: "spells",
						Message: fmt.Sprintf("target spell 0x%08X has no known presentation name in game catalog", tgt.BaseItemID),
					})
					continue
				}
				resolvedName := res.Item.Presentation.Name.Value

				if _, dup := seenRawIDs[rawID]; dup {
					spellLoadoutValid = false
					blockingIssues = append(blockingIssues, BuildTemplatePreviewIssue{
						Code:    IssueCodeInvalidSpellLoadout,
						Section: "spells",
						Message: fmt.Sprintf("target spell 0x%08X is duplicated in compact loadout", tgt.BaseItemID),
					})
					continue
				}
				seenRawIDs[rawID] = struct{}{}
				usedMemorySlots += memCost
				resolvedEquipped = append(resolvedEquipped, buildtemplates.SpellSlotRef{
					BaseItemID: tgt.BaseItemID,
					Name:       resolvedName,
				})
				resolvedRawIDs = append(resolvedRawIDs, rawID)
			}

			if usedMemorySlots > 12 {
				spellLoadoutValid = false
				blockingIssues = append(blockingIssues, BuildTemplatePreviewIssue{
					Code:    IssueCodeInvalidSpellLoadout,
					Section: "spells",
					Message: fmt.Sprintf("used memory slots %d exceeds maximum capacity 12", usedMemorySlots),
				})
			}
			if rawSpells.AvailableMemorySlots > 0 && usedMemorySlots > rawSpells.AvailableMemorySlots {
				spellLoadoutValid = false
				blockingIssues = append(blockingIssues, BuildTemplatePreviewIssue{
					Code:    IssueCodeInvalidSpellLoadout,
					Section: "spells",
					Message: fmt.Sprintf("used memory slots %d exceeds character available memory slots %d", usedMemorySlots, rawSpells.AvailableMemorySlots),
				})
			}

			// Step 9.4: Build Slots comparisons against the validated compact target state only if loadout is valid.
			if spellLoadoutValid {
				spPlan := &SpellsPreviewPlan{
					UsedMemorySlots:      usedMemorySlots,
					AvailableMemorySlots: rawSpells.AvailableMemorySlots,
					EquippedSpells:       resolvedEquipped,
				}
				for i := 0; i < 12; i++ {
					var curRef *buildtemplates.SpellSlotRef
					if i < len(spellsRes.Spells) {
						s := spellsRes.Spells[i]
						if s.RawMagicParamID != 0 && s.RawMagicParamID != 0xFFFFFFFF {
							curRef = &buildtemplates.SpellSlotRef{
								BaseItemID: buildtemplates.SpellItemIDPrefix | s.RawMagicParamID,
								Name:       s.Name,
							}
						}
					}
					var tgtRef *buildtemplates.SpellSlotRef
					if i < len(resolvedEquipped) {
						refCopy := resolvedEquipped[i]
						tgtRef = &refCopy
					}

					changed := false
					if curRef == nil && tgtRef != nil {
						changed = true
					} else if curRef != nil && tgtRef == nil {
						changed = true
					} else if curRef != nil && tgtRef != nil {
						changed = (curRef.BaseItemID != tgtRef.BaseItemID)
					}

					spPlan.Slots = append(spPlan.Slots, SpellSlotChange{
						SlotNumber: i + 1,
						Current:    curRef,
						Target:     tgtRef,
						Changed:    changed,
					})
				}
				plan.Spells = spPlan
				targetSpells = &saveengine.CharacterSpellsPlan{
					RawMagicParamIDs: resolvedRawIDs,
					UsedMemorySlots:  usedMemorySlots,
				}
			}
		}
	}

	// 10. Final save read consistency check.
	finalUndo, err := engine.GetUndoState(saveSessionID, characterID)
	if err != nil {
		return resolvedBuildTemplatePlan{}, err
	}
	if finalUndo.SaveRevision != initialUndo.SaveRevision {
		return resolvedBuildTemplatePlan{}, ErrSaveRevisionConflict
	}

	executable := len(blockingIssues) == 0
	if !executable {
		targetName = nil
		targetAttrs = nil
		targetSpells = nil
	}

	return resolvedBuildTemplatePlan{
		previewResult: GetBuildTemplatePreviewResult{
			TemplateID:       templateID,
			TemplateRevision: templateRevision,
			CharacterID:      characterID,
			SaveSessionID:    saveSessionID,
			SaveRevision:     initialUndo.SaveRevision,
			Executable:       executable,
			Plan:             plan,
			BlockingIssues:   blockingIssues,
		},
		targetName:   targetName,
		targetAttrs:  targetAttrs,
		targetSpells: targetSpells,
	}, nil
}

func formatBlockingIssues(issues []BuildTemplatePreviewIssue) string {
	if len(issues) == 0 {
		return ""
	}
	parts := make([]string, len(issues))
	for i, iss := range issues {
		parts[i] = fmt.Sprintf("[%s] %s", iss.Code, iss.Message)
	}
	return strings.Join(parts, "; ")
}
