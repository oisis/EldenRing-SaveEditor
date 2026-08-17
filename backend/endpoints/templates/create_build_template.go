/*
Endpoint: CreateBuildTemplate
EndpointID: create_build_template
Purpose: Creates a new Build Template in the local templates library from an explicit validated data selection.
How it works: The runtime handler verifies the save session revision before and after reading character data, resolves and builds the requested schema v2 template sections (Profile name/level, Stats, Spells 1-12), validates the complete BuildTemplate document, and delegates persistence to the local templates store.
Supported resource types: GameResource references.
Input variables: saveSessionID, sourceCharacterID, selection, name, description, tags.
GameCatalog variables read: for occupied spell memory slots (1-12), the resource name and cost from GameCatalog.
Save variables processed: the UserData10 activity flag and profile of the source character, stats attributes, and equipped spells; the endpoint is non-mutating on save state.
Implementation status: implemented
*/
package templates

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/oisis/EldenRing-SaveForge/backend/buildtemplates"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/equipment"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// CreateBuildTemplateEndpointID is the stable backend identifier of CreateBuildTemplate.
const CreateBuildTemplateEndpointID = "create_build_template"

// CreateBuildTemplateDefinition describes the public mutation contract.
var CreateBuildTemplateDefinition = contract.MustDefine(contract.Definition{
	Name:                       "CreateBuildTemplate",
	ID:                         CreateBuildTemplateEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "GameResource references",
	SupportedResourceVariables: []string{"saveSessionID", "sourceCharacterID", "selection", "name", "description", "tags"},
	Description:                "Creates a new Build Template from an explicit validated data selection.",
})

// ErrSaveRevisionConflict indicates that the save revision changed while reading
// character data to construct the template document; nothing was written.
var ErrSaveRevisionConflict = errors.New("save revision changed during template creation")

// CreateBuildTemplateRequest is the typed request body for CreateBuildTemplate.
type CreateBuildTemplateRequest struct {
	SaveSessionID     string                           `json:"saveSessionID"`
	SourceCharacterID int                              `json:"sourceCharacterID"`
	Selection         buildtemplates.TemplateSelection `json:"selection"`
	Name              string                           `json:"name"`
	Description       string                           `json:"description,omitempty"`
	Tags              []string                         `json:"tags,omitempty"`
}

// CreateBuildTemplateResult is the typed receipt of CreateBuildTemplate.
type CreateBuildTemplateResult struct {
	TemplateID       string `json:"templateID"`
	TemplateRevision string `json:"templateRevision"`
}

// CreateBuildTemplate creates a new schema version 2 Build Template from an active
// save character and persists it to the local templates library.
func CreateBuildTemplate(
	store *buildtemplates.Store,
	engine *saveengine.Engine,
	catalog *gamecatalog.Catalog,
	appVersion string,
	req CreateBuildTemplateRequest,
) (CreateBuildTemplateResult, error) {
	if store == nil {
		return CreateBuildTemplateResult{}, errors.New("templates store is not available")
	}
	if engine == nil {
		return CreateBuildTemplateResult{}, errors.New("save engine is not available")
	}
	if catalog == nil {
		return CreateBuildTemplateResult{}, errors.New("game catalog is not available")
	}
	if req.SaveSessionID == "" {
		return CreateBuildTemplateResult{}, errors.New("saveSessionID is required")
	}
	if req.SourceCharacterID < 0 || req.SourceCharacterID >= 10 {
		return CreateBuildTemplateResult{}, fmt.Errorf("sourceCharacterID %d is outside the range 0..9", req.SourceCharacterID)
	}
	if strings.TrimSpace(req.Name) == "" {
		return CreateBuildTemplateResult{}, errors.New("name must not be empty")
	}
	if !req.Selection.HasAnySelected() {
		return CreateBuildTemplateResult{}, errors.New("selection must have at least one field selected")
	}

	// Validate supported sections and reject unsupported selections fail-closed.
	if req.Selection.InventoryWorkspace.HasAny() {
		return CreateBuildTemplateResult{}, errors.New("selection.inventory.workspace is not supported in schema v2 template creation")
	}
	if req.Selection.Equipment.HasAny() {
		return CreateBuildTemplateResult{}, errors.New("selection.equipment is not supported in this version")
	}
	if req.Selection.Items.HasAny() {
		return CreateBuildTemplateResult{}, errors.New("selection.items is not supported in this version")
	}
	if req.Selection.InventoryLayout.HasAny() {
		return CreateBuildTemplateResult{}, errors.New("selection.inventoryLayout is not supported in this version")
	}
	if req.Selection.StorageLayout.HasAny() {
		return CreateBuildTemplateResult{}, errors.New("selection.storageLayout is not supported in this version")
	}

	if req.Selection.Profile != nil {
		if req.Selection.Profile.All {
			return CreateBuildTemplateResult{}, errors.New("selection.profile: boolean shortcut (All) is not supported because unconfirmed profile fields cannot be exported; select specific fields (name, level)")
		}
		for field, selected := range req.Selection.Profile.Fields {
			if !selected {
				continue
			}
			if field != "name" && field != "level" {
				return CreateBuildTemplateResult{}, fmt.Errorf("selection.profile.%s is not supported in template export (only name and level are supported)", field)
			}
		}
	}

	if req.Selection.Stats != nil && req.Selection.Stats.Fields != nil {
		for field, selected := range req.Selection.Stats.Fields {
			if !selected {
				continue
			}
			switch field {
			case "vigor", "mind", "endurance", "strength", "dexterity", "intelligence", "faith", "arcane":
			default:
				return CreateBuildTemplateResult{}, fmt.Errorf("selection.stats.%s is unknown", field)
			}
		}
	}

	if req.Selection.Spells != nil {
		if req.Selection.Spells.All {
			return CreateBuildTemplateResult{}, errors.New("selection.spells: boolean shortcut (All) is not supported because unconfirmed slots (spell13, spell14) cannot be exported; select specific slots (spell1..spell12)")
		}
		for field, selected := range req.Selection.Spells.Fields {
			if !selected {
				continue
			}
			if field == "spell13" || field == "spell14" {
				return CreateBuildTemplateResult{}, errors.New("selection.spells: spell13 and spell14 are not supported (public contract supports slots 1-12)")
			}
		}
	}

	// 1. Initial undo state check (proves session existence and captures initial saveRevision).
	initialUndo, err := engine.GetUndoState(req.SaveSessionID, req.SourceCharacterID)
	if err != nil {
		return CreateBuildTemplateResult{}, err
	}

	// 2. Read character profile and verify slot is active.
	profile, err := engine.GetCharacterProfile(req.SaveSessionID, req.SourceCharacterID)
	if err != nil {
		return CreateBuildTemplateResult{}, err
	}
	if !profile.Active {
		return CreateBuildTemplateResult{}, fmt.Errorf("character slot %d is inactive", req.SourceCharacterID)
	}

	// 3. Build selected sections.
	var sections buildtemplates.TemplateSections

	if req.Selection.Profile != nil && req.Selection.Profile.HasAny() {
		pSec := &buildtemplates.ProfileSection{}
		if req.Selection.Profile.Fields["name"] {
			nameVal := profile.Name
			pSec.Name = &nameVal
		}
		if req.Selection.Profile.Fields["level"] {
			levelVal := profile.Level
			pSec.Level = &levelVal
		}
		sections.Profile = pSec
	}

	if req.Selection.Stats != nil && req.Selection.Stats.HasAny() {
		stats, err := engine.GetCharacterStats(req.SaveSessionID, req.SourceCharacterID)
		if err != nil {
			return CreateBuildTemplateResult{}, err
		}
		sSec := &buildtemplates.StatsSection{}
		selAll := req.Selection.Stats.All
		fields := req.Selection.Stats.Fields
		if selAll || fields["vigor"] {
			v := stats.Vigor
			sSec.Vigor = &v
		}
		if selAll || fields["mind"] {
			v := stats.Mind
			sSec.Mind = &v
		}
		if selAll || fields["endurance"] {
			v := stats.Endurance
			sSec.Endurance = &v
		}
		if selAll || fields["strength"] {
			v := stats.Strength
			sSec.Strength = &v
		}
		if selAll || fields["dexterity"] {
			v := stats.Dexterity
			sSec.Dexterity = &v
		}
		if selAll || fields["intelligence"] {
			v := stats.Intelligence
			sSec.Intelligence = &v
		}
		if selAll || fields["faith"] {
			v := stats.Faith
			sSec.Faith = &v
		}
		if selAll || fields["arcane"] {
			v := stats.Arcane
			sSec.Arcane = &v
		}
		sections.Stats = sSec
	}

	if req.Selection.Spells != nil && req.Selection.Spells.HasAny() {
		spellsRes, err := equipment.GetEquippedSpells(engine, catalog, req.SaveSessionID, req.SourceCharacterID)
		if err != nil {
			return CreateBuildTemplateResult{}, err
		}
		spSec := &buildtemplates.SpellsSection{}
		fields := req.Selection.Spells.Fields

		slotRef := func(index int) *buildtemplates.SpellSlotRef {
			if index >= len(spellsRes.Spells) {
				return nil
			}
			slot := spellsRes.Spells[index]
			if slot.RawMagicParamID == 0xFFFFFFFF || slot.RawMagicParamID == 0 {
				return nil
			}
			baseItemID := buildtemplates.SpellItemIDPrefix | slot.RawMagicParamID
			return &buildtemplates.SpellSlotRef{
				BaseItemID: baseItemID,
				Name:       slot.Name,
			}
		}

		if fields["spell1"] {
			spSec.Spell1 = slotRef(0)
		}
		if fields["spell2"] {
			spSec.Spell2 = slotRef(1)
		}
		if fields["spell3"] {
			spSec.Spell3 = slotRef(2)
		}
		if fields["spell4"] {
			spSec.Spell4 = slotRef(3)
		}
		if fields["spell5"] {
			spSec.Spell5 = slotRef(4)
		}
		if fields["spell6"] {
			spSec.Spell6 = slotRef(5)
		}
		if fields["spell7"] {
			spSec.Spell7 = slotRef(6)
		}
		if fields["spell8"] {
			spSec.Spell8 = slotRef(7)
		}
		if fields["spell9"] {
			spSec.Spell9 = slotRef(8)
		}
		if fields["spell10"] {
			spSec.Spell10 = slotRef(9)
		}
		if fields["spell11"] {
			spSec.Spell11 = slotRef(10)
		}
		if fields["spell12"] {
			spSec.Spell12 = slotRef(11)
		}
		sections.Spells = spSec
	}

	// 4. Verify save revision consistency after reading all sections.
	finalUndo, err := engine.GetUndoState(req.SaveSessionID, req.SourceCharacterID)
	if err != nil {
		return CreateBuildTemplateResult{}, err
	}
	if finalUndo.SaveRevision != initialUndo.SaveRevision {
		return CreateBuildTemplateResult{}, ErrSaveRevisionConflict
	}

	// 5. Construct BuildTemplate document.
	nowUTC := time.Now().UTC().Format(time.RFC3339Nano)
	var tagsCopy []string
	if len(req.Tags) > 0 {
		tagsCopy = append([]string(nil), req.Tags...)
	}

	tpl := &buildtemplates.BuildTemplate{
		Schema:     buildtemplates.SchemaKey,
		Version:    buildtemplates.MaxSchemaVersion,
		CreatedAt:  nowUTC,
		AppVersion: appVersion,
		Metadata: &buildtemplates.TemplateDocMetadata{
			Name:                 req.Name,
			Description:          req.Description,
			Tags:                 tagsCopy,
			SourceCharacterIndex: req.SourceCharacterID,
			SourceCharacterName:  profile.Name,
		},
		Selection: &req.Selection,
		Sections:  sections,
	}

	if err := buildtemplates.ValidateTemplate(tpl); err != nil {
		return CreateBuildTemplateResult{}, fmt.Errorf("validate generated template: %w", err)
	}

	// 6. Persist to store library.
	templateID, templateRevision, err := store.CreateTemplate(tpl)
	if err != nil {
		return CreateBuildTemplateResult{}, err
	}

	return CreateBuildTemplateResult{
		TemplateID:       templateID,
		TemplateRevision: templateRevision,
	}, nil
}
