/*
Endpoint: GetSafetyProfile
EndpointID: get_safety_profile
Purpose: Returns the global Safety Profile of the host application together with the closed list of profiles it accepts.
How it works: The runtime handler reads the host-local application settings store, which resolves the persisted profile once and falls back to the product default only when the host never stored one. It touches no save session, no snapshot and no GameCatalog document.
Supported resource types: —.
Input variables: none.
GameCatalog variables read: none.
Save variables read: none; the profile is an application setting and is never part of a save snapshot.
Implementation status: implemented
*/
package application

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/safetyprofile"
)

// GetSafetyProfileEndpointID is the stable backend identifier of GetSafetyProfile.
const GetSafetyProfileEndpointID = "get_safety_profile"

// GetSafetyProfileDefinition describes the public getter contract.
var GetSafetyProfileDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetSafetyProfile",
	ID:                         GetSafetyProfileEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: nil,
	Description:                "Returns the global Safety Profile of the host application together with the closed list of profiles it accepts.",
})

// SafetyProfileResult is the shared wire shape of the safety-profile getter and
// setter. Both report exactly the profile now in effect plus the closed
// vocabulary, so a client never has to hardcode the three values and never has
// to interpret them: which limits a profile applies and which resources it
// reveals stays a backend decision.
type SafetyProfileResult struct {
	SafetyProfile     string   `json:"safetyProfile"`
	AvailableProfiles []string `json:"availableProfiles"`
	DefaultProfile    string   `json:"defaultProfile"`
}

// GetSafetyProfile reports the profile the backend currently enforces.
//
// store is a backend dependency supplied by the composition root, not a client
// parameter. A missing store is a wiring error rather than a client error, and
// it is never replaced by an implicit in-memory default here: a caller that
// received a profile must be able to rely on it having been read from the same
// place the setter writes.
func GetSafetyProfile(store *safetyprofile.Store) (SafetyProfileResult, error) {
	if store == nil {
		return SafetyProfileResult{}, errors.New("application settings are not available")
	}
	profile, err := store.Get()
	if err != nil {
		return SafetyProfileResult{}, err
	}
	return safetyProfileResult(profile), nil
}

// safetyProfileResult renders one profile together with the closed vocabulary.
// The slice is built per call, so a caller mutating one result cannot affect
// another call.
func safetyProfileResult(profile safetyprofile.Profile) SafetyProfileResult {
	available := make([]string, 0, len(safetyprofile.Profiles()))
	for _, value := range safetyprofile.Profiles() {
		available = append(available, string(value))
	}
	return SafetyProfileResult{
		SafetyProfile:     string(profile),
		AvailableProfiles: available,
		DefaultProfile:    string(safetyprofile.Default),
	}
}
