/*
Endpoint: SetSaveLifecycleSettings
EndpointID: set_save_lifecycle_settings
Purpose: Updates the host-local automatic-backup retention limit and backup name pattern.
How it works: SaveEngine validates the retention limit and the backup name pattern and atomically persists both without changing any save revision. The pattern accepts exactly the tokens {filename} and {timestamp}, each once, plus safe literal text; an empty pattern restores the default. An unknown token, a path separator, a control character or a name that is not portable between the supported systems is rejected. Changing the pattern renames no existing backup.
Supported resource types: —.
Input variables: backupRetention, backupNamePattern.
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
	SupportedResourceTypes: "—", SupportedResourceVariables: []string{"backupRetention", "backupNamePattern"},
	Description: "Updates the host-local automatic-backup retention limit and backup name pattern.",
})

type SetSaveLifecycleSettingsResult = saveengine.SaveLifecycleSettings

func SetSaveLifecycleSettings(
	engine *saveengine.Engine, backupRetention int, backupNamePattern string,
) (SetSaveLifecycleSettingsResult, error) {
	if engine == nil {
		return SetSaveLifecycleSettingsResult{}, errors.New("save engine is not available")
	}
	return engine.SetSaveLifecycleSettings(backupRetention, backupNamePattern)
}
