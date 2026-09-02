/*
Endpoint: SetSaveLifecycleSettings
EndpointID: set_save_lifecycle_settings
Purpose: Updates the host-local automatic-backup retention limit.
How it works: SaveEngine validates and atomically persists the setting without changing any save revision.
Supported resource types: —.
Input variables: backupRetention.
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

const SetSaveLifecycleSettingsEndpointID = "set_save_lifecycle_settings"

var SetSaveLifecycleSettingsDefinition = contract.MustDefine(contract.Definition{
	Name: "SetSaveLifecycleSettings", ID: SetSaveLifecycleSettingsEndpointID, Kind: contract.Mutation,
	SupportedResourceTypes: "—", SupportedResourceVariables: []string{"backupRetention"},
	Description: "Updates the host-local automatic-backup retention limit.",
})

type SetSaveLifecycleSettingsResult = saveengine.SaveLifecycleSettings

func SetSaveLifecycleSettings(engine *saveengine.Engine, backupRetention int) (SetSaveLifecycleSettingsResult, error) {
	if engine == nil {
		return SetSaveLifecycleSettingsResult{}, errors.New("save engine is not available")
	}
	return engine.SetSaveLifecycleSettings(backupRetention)
}
