/*
Endpoint: ExportRecoveryJournal
EndpointID: export_recovery_journal
Purpose: Copies one protected recovery journal to an explicit diagnostic export target.
How it works: SaveEngine validates the journal identifier and atomically writes an independent user-selected copy without applying it.
Supported resource types: —.
Input variables: journalID, target.
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

const ExportRecoveryJournalEndpointID = "export_recovery_journal"

var ExportRecoveryJournalDefinition = contract.MustDefine(contract.Definition{
	Name: "ExportRecoveryJournal", ID: ExportRecoveryJournalEndpointID, Kind: contract.Mutation,
	SupportedResourceTypes: "—", SupportedResourceVariables: []string{"journalID", "target"},
	Description: "Copies one protected recovery journal to an explicit diagnostic export target.",
})

func ExportRecoveryJournal(engine *saveengine.Engine, journalID, target string) error {
	if engine == nil {
		return errors.New("save engine is not available")
	}
	return engine.ExportRecoveryJournal(journalID, target)
}
