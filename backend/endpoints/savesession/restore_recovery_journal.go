/*
Endpoint: RestoreRecoveryJournal
EndpointID: restore_recovery_journal
Purpose: Rebuilds a new in-memory session from a compatible recovery journal.
How it works: SaveEngine reloads the unchanged source, verifies its fingerprint, replays every retained operation in order and validates the result before registration.
Supported resource types: —.
Input variables: journalID.
GameCatalog variables read: none.
Save variables processed: a new private session snapshot; the source file is never written.
Implementation status: implemented
*/
package savesession

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

const RestoreRecoveryJournalEndpointID = "restore_recovery_journal"

var RestoreRecoveryJournalDefinition = contract.MustDefine(contract.Definition{
	Name: "RestoreRecoveryJournal", ID: RestoreRecoveryJournalEndpointID, Kind: contract.Mutation,
	SupportedResourceTypes: "—", SupportedResourceVariables: []string{"journalID"},
	Description: "Rebuilds a new in-memory session from a compatible recovery journal.",
})

type RestoreRecoveryJournalResult = saveengine.SessionInfo

func RestoreRecoveryJournal(engine *saveengine.Engine, journalID string) (RestoreRecoveryJournalResult, error) {
	if engine == nil {
		return RestoreRecoveryJournalResult{}, errors.New("save engine is not available")
	}
	return engine.RestoreRecoveryJournal(journalID)
}
