package application

import (
	"reflect"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/safetyprofile"
)

// The getter and the setter share one wire shape, so a client refreshes its
// cached profile from the answer of the call that changed it. This file covers
// that public contract; the persistence rules belong to backend/safetyprofile.

func wantAvailableProfiles() []string {
	available := make([]string, 0, len(safetyprofile.Profiles()))
	for _, profile := range safetyprofile.Profiles() {
		available = append(available, string(profile))
	}
	return available
}

func TestSafetyProfileGetterAndSetterShareOneResult(t *testing.T) {
	store := safetyprofile.NewStore(t.TempDir())

	initial, err := GetSafetyProfile(store)
	if err != nil {
		t.Fatalf("GetSafetyProfile: %v", err)
	}
	want := SafetyProfileResult{
		SafetyProfile:     string(safetyprofile.Default),
		AvailableProfiles: wantAvailableProfiles(),
		DefaultProfile:    string(safetyprofile.Default),
	}
	if !reflect.DeepEqual(initial, want) {
		t.Errorf("GetSafetyProfile = %+v, want %+v", initial, want)
	}

	set, err := SetSafetyProfile(store, string(safetyprofile.Chaos))
	if err != nil {
		t.Fatalf("SetSafetyProfile: %v", err)
	}
	want.SafetyProfile = string(safetyprofile.Chaos)
	if !reflect.DeepEqual(set, want) {
		t.Errorf("SetSafetyProfile = %+v, want %+v", set, want)
	}

	// The getter reports what the setter stored, so the two never disagree.
	after, err := GetSafetyProfile(store)
	if err != nil {
		t.Fatalf("GetSafetyProfile after the change: %v", err)
	}
	if !reflect.DeepEqual(after, set) {
		t.Errorf("GetSafetyProfile = %+v, want the setter's %+v", after, set)
	}

	// The returned slice is built per call, so mutating one answer cannot reach
	// another.
	after.AvailableProfiles[0] = "tampered"
	fresh, err := GetSafetyProfile(store)
	if err != nil {
		t.Fatalf("GetSafetyProfile: %v", err)
	}
	if !reflect.DeepEqual(fresh.AvailableProfiles, wantAvailableProfiles()) {
		t.Errorf("availableProfiles = %v, want %v",
			fresh.AvailableProfiles, wantAvailableProfiles())
	}
}

// An unknown value is rejected and leaves the stored profile in effect.
func TestSetSafetyProfileRejectsAnUnknownValue(t *testing.T) {
	store := safetyprofile.NewStore(t.TempDir())
	if _, err := SetSafetyProfile(store, string(safetyprofile.ExpandedLimits)); err != nil {
		t.Fatalf("SetSafetyProfile: %v", err)
	}

	result, err := SetSafetyProfile(store, "chaos_mode")
	if err == nil {
		t.Fatalf("an unknown profile was accepted: %+v", result)
	}
	if !reflect.DeepEqual(result, SetSafetyProfileResult{}) {
		t.Errorf("result = %+v, want the zero value", result)
	}
	current, err := GetSafetyProfile(store)
	if err != nil {
		t.Fatalf("GetSafetyProfile: %v", err)
	}
	if current.SafetyProfile != string(safetyprofile.ExpandedLimits) {
		t.Errorf("the refused write changed the profile to %q, want %q",
			current.SafetyProfile, safetyprofile.ExpandedLimits)
	}
}

// A missing store is a wiring error, never an implicit in-memory default: a
// caller that received a profile must be able to rely on where it came from.
func TestSafetyProfileEndpointsRejectAMissingStore(t *testing.T) {
	if result, err := GetSafetyProfile(nil); err == nil {
		t.Errorf("GetSafetyProfile accepted a missing store: %+v", result)
	}
	if result, err := SetSafetyProfile(nil, string(safetyprofile.Safe)); err == nil {
		t.Errorf("SetSafetyProfile accepted a missing store: %+v", result)
	}
}

func TestSafetyProfileContractsDeclareTheirVariables(t *testing.T) {
	if GetSafetyProfileDefinition.SupportedResourceVariables != nil {
		t.Errorf("GetSafetyProfile variables = %v, want none",
			GetSafetyProfileDefinition.SupportedResourceVariables)
	}
	want := []string{"safetyProfile"}
	if !reflect.DeepEqual(SetSafetyProfileDefinition.SupportedResourceVariables, want) {
		t.Errorf("SetSafetyProfile variables = %v, want %v",
			SetSafetyProfileDefinition.SupportedResourceVariables, want)
	}
}
