/*
Endpoint: GetRecoveryJournal
EndpointID: get_recovery_journal
Purpose: Returns the safe inspection view of one recovery journal.
How it works: SaveEngine validates the protected journal and returns operation metadata without replay bytes.
Supported resource types: —.
Input variables: journalID.
GameCatalog variables read: none.
Save variables read: source bytes only for fingerprint comparison.
Implementation status: implemented
*/
package savesession

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

const GetRecoveryJournalEndpointID = "get_recovery_journal"

var GetRecoveryJournalDefinition = contract.MustDefine(contract.Definition{
	Name: "GetRecoveryJournal", ID: GetRecoveryJournalEndpointID, Kind: contract.Getter,
	SupportedResourceTypes: "—", SupportedResourceVariables: []string{"journalID"},
	Description: "Returns the safe inspection view of one recovery journal.",
})

type GetRecoveryJournalResult = saveengine.RecoveryJournalSummary

func GetRecoveryJournal(engine *saveengine.Engine, journalID string) (GetRecoveryJournalResult, error) {
	if engine == nil {
		return GetRecoveryJournalResult{}, errors.New("save engine is not available")
	}
	return engine.GetRecoveryJournal(journalID)
}
