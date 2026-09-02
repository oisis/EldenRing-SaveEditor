/*
Endpoint: DiscardRecoveryJournal
EndpointID: discard_recovery_journal
Purpose: Deletes one exact protected recovery journal without touching its source save.
How it works: SaveEngine validates the identifier and removes only its own journal file.
Supported resource types: —.
Input variables: journalID.
GameCatalog variables read: none.
Save variables processed: none.
Implementation status: implemented
*/
package savesession

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

const DiscardRecoveryJournalEndpointID = "discard_recovery_journal"

var DiscardRecoveryJournalDefinition = contract.MustDefine(contract.Definition{
	Name: "DiscardRecoveryJournal", ID: DiscardRecoveryJournalEndpointID, Kind: contract.Mutation,
	SupportedResourceTypes: "—", SupportedResourceVariables: []string{"journalID"},
	Description: "Deletes one exact protected recovery journal without touching its source save.",
})

func DiscardRecoveryJournal(engine *saveengine.Engine, journalID string) error {
	if engine == nil {
		return errors.New("save engine is not available")
	}
	return engine.DiscardRecoveryJournal(journalID)
}
