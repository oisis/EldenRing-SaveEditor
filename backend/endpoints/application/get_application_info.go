/*
Endpoint: GetApplicationInfo
EndpointID: get_application_info
Purpose: Zwraca wersję aplikacji, wersje obsługiwanych schematów oraz podstawowe informacje o możliwościach backendu.
How it works: The runtime handler validates the application version supplied by its backend caller and returns it together with the compile-time GameCatalog schema version range and the capabilities the backend currently declares. It reads no catalog instance, no manifest and no save.
Supported resource types: —.
Input variables: none.
GameCatalog variables read: only the MinimumSchemaVersion and CurrentSchemaVersion constants.
Save variables read: none.
Implementation status: implemented.
*/
package application

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

// GetApplicationInfoEndpointID is the stable backend identifier of GetApplicationInfo.
const GetApplicationInfoEndpointID = "get_application_info"

// gameCatalogSchemaName is the name of the only schema the backend supports today.
const gameCatalogSchemaName = "game_catalog"

// catalogReadCapability is the only capability the backend declares today. Save
// reading, save writing and mutations are not declared yet.
// ponytail: a plain constant, not a capability registry; one value needs no model.
const catalogReadCapability = "catalog_read"

// GetApplicationInfoDefinition describes the public getter contract.
var GetApplicationInfoDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetApplicationInfo",
	ID:                         GetApplicationInfoEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: nil,
	Description:                "Zwraca wersję aplikacji, wersje obsługiwanych schematów oraz podstawowe informacje o możliwościach backendu.",
})

// SupportedSchema reports one schema the backend can read and the version range
// it accepts for it.
type SupportedSchema struct {
	Name           string `json:"name"`
	MinimumVersion uint32 `json:"minimumVersion"`
	CurrentVersion uint32 `json:"currentVersion"`
}

// GetApplicationInfoResult is the typed result of GetApplicationInfo.
type GetApplicationInfoResult struct {
	ApplicationVersion string            `json:"applicationVersion"`
	SupportedSchemas   []SupportedSchema `json:"supportedSchemas"`
	Capabilities       []string          `json:"capabilities"`
}

// GetApplicationInfo reports the application version together with the schema
// versions and capabilities of the current backend.
//
// applicationVersion is a backend dependency, not a client parameter: the
// backend caller owns the single source of the application version and passes
// it in. The endpoint returns it exactly as supplied — it is never trimmed,
// normalised, or replaced by a fallback — and it reads no build file, generated
// version file, or environment variable of its own. An empty version is a
// backend wiring error, not a client error.
func GetApplicationInfo(applicationVersion string) (GetApplicationInfoResult, error) {
	if applicationVersion == "" {
		return GetApplicationInfoResult{}, errors.New("application version is required")
	}

	// Both slices are built per call, so a caller mutating one result cannot
	// affect another call.
	return GetApplicationInfoResult{
		ApplicationVersion: applicationVersion,
		SupportedSchemas: []SupportedSchema{{
			Name:           gameCatalogSchemaName,
			MinimumVersion: schema.MinimumSchemaVersion,
			CurrentVersion: schema.CurrentSchemaVersion,
		}},
		Capabilities: []string{catalogReadCapability},
	}, nil
}
