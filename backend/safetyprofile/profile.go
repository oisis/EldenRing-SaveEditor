// Package safetyprofile owns the single backend interpretation of the global
// SaveForge Safety Profile: which resource limits apply, which resources the
// general Item Database may present, and which additions need an explicit
// ban-risk confirmation.
//
// The profile is a host application setting. It belongs to no save session, it
// is never part of a save snapshot, and SaveEngine neither reads nor stores it.
// Every consumer — catalog getters, item mutations and the desktop bridge —
// resolves its decisions through this package, so no getter, mutation, bridge
// method or frontend module reimplements the rules below.
package safetyprofile

import "fmt"

// Profile is the closed vocabulary of the global safety setting. The values are
// the exact wire strings; they are never trimmed, recased or aliased.
type Profile string

const (
	// Safe uses the Safe Mode limit of a resource where the catalog states one
	// and hides every ban-risk or cut-content resource.
	Safe Profile = "safe"
	// ExpandedLimits raises the limits to the base catalog limits and keeps the
	// same visibility rules as Safe.
	ExpandedLimits Profile = "expanded_limits"
	// Chaos uses the base catalog limits and additionally reveals ban-risk and
	// cut-content resources.
	Chaos Profile = "chaos"
)

// Default is the profile a host that never stored one runs under.
const Default = Safe

// Profiles is the canonical order the three profiles are reported in.
func Profiles() []Profile {
	return []Profile{Safe, ExpandedLimits, Chaos}
}

// Parse accepts exactly one of the three profile values. There is deliberately
// no empty form and no alias: a caller states which profile it means, and an
// unknown value is rejected instead of silently becoming the default.
func Parse(value string) (Profile, error) {
	switch Profile(value) {
	case Safe:
		return Safe, nil
	case ExpandedLimits:
		return ExpandedLimits, nil
	case Chaos:
		return Chaos, nil
	}
	return "", fmt.Errorf(
		"unknown safety profile %q; expected %q, %q or %q", value, Safe, ExpandedLimits, Chaos)
}
