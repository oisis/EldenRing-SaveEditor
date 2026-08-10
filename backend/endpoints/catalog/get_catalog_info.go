/*
Endpoint: GetCatalogInfo
EndpointID: get_catalog_info
Purpose: Returns the GameCatalog schema and data versions, game version, validation status, and the manifest of sources used.
How it works: The runtime handler reads the manifest of the already loaded GameCatalog, checks it with the existing schema.ValidateManifest validator and returns it as a typed result without loading, reloading or modifying the catalog.
Supported resource types: —.
Input variables: none.
GameCatalog variables read: manifest schemaVersion, dataVersion, gameVersion and sources.
Save variables read: none; the endpoint never opens or reads a save.
Implementation status: implemented; GetCatalogInfo is the runtime handler of this contract.
*/
package catalog

import (
	"errors"
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

// GetCatalogInfoEndpointID is the stable backend identifier of GetCatalogInfo.
const GetCatalogInfoEndpointID = "get_catalog_info"

// GetCatalogInfoDefinition describes the public getter contract.
var GetCatalogInfoDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetCatalogInfo",
	ID:                         GetCatalogInfoEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: nil,
	Description:                "Returns the GameCatalog schema and data versions, game version, validation status, and the manifest of sources used.",
})

// GetCatalogInfoResult is the typed result of GetCatalogInfo.
type GetCatalogInfoResult struct {
	SchemaVersion uint32              `json:"schemaVersion"`
	DataVersion   string              `json:"dataVersion"`
	GameVersion   string              `json:"gameVersion"`
	Valid         bool                `json:"valid"`
	Sources       []schema.DataSource `json:"sources"`
}

// GetCatalogInfo reports the manifest of the loaded GameCatalog. Because
// gamecatalog.Catalog is an exported type, a caller can pass a zero-value
// catalog that never went through gamecatalog.New, so the manifest is checked
// with the existing schema.ValidateManifest validator instead of being trusted.
// This endpoint defines no validation rules of its own; Valid=true means only
// that schema.ValidateManifest accepted the manifest.
func GetCatalogInfo(gameCatalog *gamecatalog.Catalog) (GetCatalogInfoResult, error) {
	if gameCatalog == nil {
		return GetCatalogInfoResult{}, errors.New("game catalog is not loaded")
	}

	manifest := gameCatalog.Manifest()
	if _, err := schema.ValidateManifest(manifest); err != nil {
		return GetCatalogInfoResult{}, fmt.Errorf("game catalog manifest is invalid: %w", err)
	}

	return GetCatalogInfoResult{
		SchemaVersion: manifest.SchemaVersion,
		DataVersion:   manifest.DataVersion,
		GameVersion:   manifest.GameVersion,
		Valid:         true,
		Sources:       manifest.Sources,
	}, nil
}
