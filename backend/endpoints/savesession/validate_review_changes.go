/*
Endpoint: ValidateReviewChanges
EndpointID: validate_review_changes
Purpose: Validates the exact current revision and issues a revision-bound authorization token for Save or Save As.
How it works: SaveEngine validates a private copy through serialization and reload, classifies operation risks and stores only an opaque revision-bound authorization.
Supported resource types: —.
Input variables: saveSessionID, expectedRevision.
GameCatalog variables read: none.
Save variables read: the private snapshot and safe operation metadata; no file is written.
Implementation status: implemented
*/
package savesession

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

const ValidateReviewChangesEndpointID = "validate_review_changes"

var ValidateReviewChangesDefinition = contract.MustDefine(contract.Definition{
	Name: "ValidateReviewChanges", ID: ValidateReviewChangesEndpointID, Kind: contract.Mutation,
	SupportedResourceTypes: "—", SupportedResourceVariables: []string{"saveSessionID", "expectedRevision"},
	Description: "Validates the exact current revision and issues a revision-bound authorization token for Save or Save As.",
})

type ValidateReviewChangesResult = saveengine.ReviewValidationResult

func ValidateReviewChanges(engine *saveengine.Engine, saveSessionID, expectedRevision string) (ValidateReviewChangesResult, error) {
	if engine == nil {
		return ValidateReviewChangesResult{}, errors.New("save engine is not available")
	}
	return engine.ValidateReviewChanges(saveSessionID, expectedRevision)
}
