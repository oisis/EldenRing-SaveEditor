/*
Endpoint: GetRecoveryJournals
EndpointID: get_recovery_journals
Purpose: Returns safe summaries of compatible, incompatible and corrupt recovery journals.
How it works: SaveEngine reads protected journal metadata and compares each source fingerprint without replaying or mutating anything.
Supported resource types: —.
Input variables: none.
GameCatalog variables read: none.
Save variables read: source bytes only for fingerprint comparison; no session is created.
Implementation status: implemented
*/
package savesession

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

const GetRecoveryJournalsEndpointID = "get_recovery_journals"

var GetRecoveryJournalsDefinition = contract.MustDefine(contract.Definition{
	Name: "GetRecoveryJournals", ID: GetRecoveryJournalsEndpointID, Kind: contract.Getter,
	SupportedResourceTypes: "—", SupportedResourceVariables: []string{},
	Description: "Returns safe summaries of compatible, incompatible and corrupt recovery journals.",
})

type GetRecoveryJournalsResult = []saveengine.RecoveryJournalSummary

func GetRecoveryJournals(engine *saveengine.Engine) (GetRecoveryJournalsResult, error) {
	if engine == nil {
		return nil, errors.New("save engine is not available")
	}
	return engine.GetRecoveryJournals()
}
