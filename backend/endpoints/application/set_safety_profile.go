/*
Endpoint: SetSafetyProfile
EndpointID: set_safety_profile
Purpose: Stores the global Safety Profile of the host application.
How it works: The runtime handler passes the requested value to the host-local application settings store, which accepts exactly one of the three known profiles and replaces the stored document atomically. It advances no save revision, touches no session and produces no mutation receipt, because the profile is an application setting rather than save data.
Supported resource types: —.
Input variables: safetyProfile.
GameCatalog variables read: none.
Save variables processed: none; the profile is an application setting and is never written into a save snapshot.
Implementation status: implemented
*/
package application

import (
	"errors"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/safetyprofile"
)

// SetSafetyProfileEndpointID is the stable backend identifier of SetSafetyProfile.
const SetSafetyProfileEndpointID = "set_safety_profile"

// SetSafetyProfileDefinition describes the public mutation contract.
var SetSafetyProfileDefinition = contract.MustDefine(contract.Definition{
	Name:                       "SetSafetyProfile",
	ID:                         SetSafetyProfileEndpointID,
	Kind:                       contract.Mutation,
	SupportedResourceTypes:     "—",
	SupportedResourceVariables: []string{"safetyProfile"},
	Description:                "Stores the global Safety Profile of the host application.",
})

// SetSafetyProfileResult is the profile now in effect, in the same shape the
// getter reports, so a client refreshes its cached value from the answer of the
// call that changed it.
type SetSafetyProfileResult = SafetyProfileResult

// SetSafetyProfile stores one of the three known profiles.
//
// safetyProfile is passed through byte for byte: it is never trimmed, recased
// or aliased, and an unknown value is rejected by the settings store before
// anything is written. A failed write leaves the previous profile in effect.
func SetSafetyProfile(
	store *safetyprofile.Store,
	safetyProfile string,
) (SetSafetyProfileResult, error) {
	if store == nil {
		return SetSafetyProfileResult{}, errors.New("application settings are not available")
	}
	profile, err := store.Set(safetyProfile)
	if err != nil {
		return SetSafetyProfileResult{}, err
	}
	return safetyProfileResult(profile), nil
}
