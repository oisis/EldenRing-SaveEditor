/*
Endpoint: GetSaveLifecycleSettings
EndpointID: get_save_lifecycle_settings
Purpose: Returns the host-local automatic-backup retention settings.
How it works: Delegates to SaveEngine's protected settings store.
Supported resource types: —.
Input variables: none.
GameCatalog variables read: none.
Save variables read: none.
Implementation status: implemented
*/
package savesession

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

const GetSaveLifecycleSettingsEndpointID = "get_save_lifecycle_settings"

var GetSaveLifecycleSettingsDefinition = contract.MustDefine(contract.Definition{
	Name: "GetSaveLifecycleSettings", ID: GetSaveLifecycleSettingsEndpointID, Kind: contract.Getter,
	SupportedResourceTypes: "—", SupportedResourceVariables: []string{},
	Description: "Returns the host-local automatic-backup retention settings.",
})

type GetSaveLifecycleSettingsResult = saveengine.SaveLifecycleSettings

func GetSaveLifecycleSettings(engine *saveengine.Engine) (GetSaveLifecycleSettingsResult, error) {
	if engine == nil {
		return GetSaveLifecycleSettingsResult{}, errors.New("save engine is not available")
	}
	return engine.GetSaveLifecycleSettings()
}
