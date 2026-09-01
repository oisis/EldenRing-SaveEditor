/*
Endpoint: ApplyBuildTemplate
EndpointID: apply_build_template
Purpose: Builds a complete plan and atomically applies the template to a character, or makes no change.
How it works: The runtime handler resolves templateID through the templates store, builds the preview plan against live save data and GameCatalog, verifies that the plan is executable and that expectedRevision matches the save revision of the plan, and delegates one atomic mutation to SaveEngine.
Supported resource types: GameResource references.
Input variables: saveSessionID, characterID, templateID, selection, options, expectedRevision.
GameCatalog variables read: for occupied spell memory slots (1-12), the presentation name and memory slot cost from GameCatalog.
Save variables processed: the active-slot flag, PlayerGameData and ProfileSummary character name, statistics attributes, recalculated level, SoulMemory, and equipped spells (slots 1-12 and active index); SaveEngine validates the complete plan and executes it as a single atomic mutation with verification and full rollback on error.
Implementation status: implemented
*/
package templates

import (
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/buildtemplates"
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// ApplyBuildTemplateEndpointID is the stable backend identifier of ApplyBuildTemplate.
const ApplyBuildTemplateEndpointID = "apply_build_template"

// ApplyBuildTemplateDefinition describes the public mutation contract.
var ApplyBuildTemplateDefinition = contract.MustDefine(contract.Definition{
	Name:                       "ApplyBuildTemplate",
	ID:                         ApplyBuildTemplateEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "GameResource references",
	SupportedResourceVariables: []string{"saveSessionID", "characterID", "templateID", "selection", "options", "expectedRevision"},
	Description:                "Builds a complete plan and atomically applies the template to a character, or makes no change.",
})

// ApplyBuildTemplateRequest is the typed request for ApplyBuildTemplate.
type ApplyBuildTemplateRequest struct {
	SaveSessionID    string                            `json:"saveSessionID"`
	CharacterID      int                               `json:"characterID"`
	TemplateID       string                            `json:"templateID"`
	Selection        *buildtemplates.TemplateSelection `json:"selection,omitempty"`
	Options          *buildtemplates.ApplyOptions      `json:"options,omitempty"`
	ExpectedRevision string                            `json:"expectedRevision"`
}

// ApplyBuildTemplateResult is the typed return of ApplyBuildTemplate. It embeds
// the receipt the central SaveEngine commit path produced, so operationKind is
// apply_build_template and never the kind of a lower writer.
type ApplyBuildTemplateResult struct {
	saveengine.MutationReceipt
	TemplateID       string                   `json:"templateID"`
	TemplateRevision string                   `json:"templateRevision"`
	CharacterID      int                      `json:"characterID"`
	Plan             BuildTemplatePreviewPlan `json:"plan"`
}

// ApplyBuildTemplate validates the build template plan against the active character state,
// checks expectedRevision, and atomically applies the plan to the character via SaveEngine.
func ApplyBuildTemplate(
	store *buildtemplates.Store,
	engine *saveengine.Engine,
	catalog *gamecatalog.Catalog,
	req ApplyBuildTemplateRequest,
) (ApplyBuildTemplateResult, error) {
	if !saveengine.IsCanonicalRevision(req.ExpectedRevision) {
		return ApplyBuildTemplateResult{}, fmt.Errorf(
			"expectedRevision must be a canonical decimal saveRevision; got %q", req.ExpectedRevision)
	}

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
		return ApplyBuildTemplateResult{}, err
	}

	if !resolved.previewResult.Executable {
		return ApplyBuildTemplateResult{}, fmt.Errorf(
			"build template plan is not executable: %s",
			formatBlockingIssues(resolved.previewResult.BlockingIssues),
		)
	}

	if req.ExpectedRevision != resolved.previewResult.SaveRevision {
		return ApplyBuildTemplateResult{}, fmt.Errorf(
			"expectedRevision %q does not match the current saveRevision %q",
			req.ExpectedRevision,
			resolved.previewResult.SaveRevision,
		)
	}

	enginePlan := saveengine.ApplyCharacterTemplatePlan{
		Name:       resolved.targetName,
		Attributes: resolved.targetAttrs,
		Spells:     resolved.targetSpells,
	}

	mutationResult, err := engine.ApplyCharacterTemplate(
		req.SaveSessionID,
		req.CharacterID,
		enginePlan,
		req.ExpectedRevision,
	)
	if err != nil {
		return ApplyBuildTemplateResult{}, err
	}

	return ApplyBuildTemplateResult{
		MutationReceipt:  mutationResult.MutationReceipt,
		TemplateID:       resolved.previewResult.TemplateID,
		TemplateRevision: resolved.previewResult.TemplateRevision,
		CharacterID:      mutationResult.CharacterID,
		Plan:             resolved.previewResult.Plan,
	}, nil
}
