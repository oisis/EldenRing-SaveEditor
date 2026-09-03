# GetSafetyProfile

## Overview

`GetSafetyProfile` returns the global **Safety Profile** of the host
application: the single setting that decides which resource limits the backend
applies and which resources it is willing to present or write.

The profile is an application setting, not save data. It belongs to no save
session, it is never part of a save snapshot, and SaveEngine neither reads nor
stores it. It is persisted per host in the application data directory, so it
survives closing a save, opening another one and restarting the application.

A host that never stored a profile runs under the product default `safe`.

| | |
|---|---|
| EndpointID | `get_safety_profile` |
| Kind | Getter |
| Domain | `application` |
| Implementation status | implemented |
| Transport status | desktop bridge only — bound as a Wails method on `desktop.Bridge` and reached by the SaveForge frontend. It is deliberately **not** registered as an HTTP route of the local explorer and therefore does not appear in `tools/swagger/openapi.json` or in the Scalar portal: an operation the explorer cannot perform must never be advertised there. |
| Implementation source | [../../../backend/endpoints/application/get_safety_profile.go](../../../backend/endpoints/application/get_safety_profile.go) |
| Policy source | [../../../backend/safetyprofile](../../../backend/safetyprofile) |
| Save access | none |

## Input

```go
func GetSafetyProfile(store *safetyprofile.Store) (SafetyProfileResult, error)
```

`store` is a backend dependency supplied by the composition root, not a client
parameter. There is no client input at all.

## Output

```go
type SafetyProfileResult struct {
	SafetyProfile     string   `json:"safetyProfile"`
	AvailableProfiles []string `json:"availableProfiles"`
	DefaultProfile    string   `json:"defaultProfile"`
}
```

`availableProfiles` is the closed vocabulary in its canonical order:
`safe`, `expanded_limits`, `chaos`. A client presents these values and caches
the active one; it never interprets them.

## Semantics of the three profiles

| Profile | Inventory limit | Storage limit | `banRisk` / `cutContent` |
|---|---|---|---|
| `safe` | `item.storage.safeModeMaxInventory` when the item declares it, otherwise `item.storage.maxInventory` | `item.storage.safeModeMaxStorage` when the item declares it, otherwise `item.storage.maxStorage` | hidden and refused |
| `expanded_limits` | `item.storage.maxInventory` | `item.storage.maxStorage` | hidden and refused |
| `chaos` | `item.storage.maxInventory` | `item.storage.maxStorage` | revealed and allowed after an explicit confirmation |

Additional rules, all owned by `backend/safetyprofile`:

- `item.safety.noDatabase` is hidden from the general Item Database under
  **every** profile, including `chaos`. Such a resource stays fully reachable
  by its exact `(kind, key)` through the feature that owns it.
- `item.safety.dlc` and `item.safety.preOrder` are presentation facts and never
  hide or block anything.
- `item.safety.banRisk` additionally requires an explicit per-operation
  confirmation in the adding flow, on top of the profile check.
- `maxInventory-sfv` and `maxStorage-sfv` are research values, never runtime
  limits, and are never read by this policy.

## Errors

| Condition | Result |
|---|---|
| the settings store is not wired | `application settings are not available` |
| the stored settings document is unreadable or carries an unknown profile | `application settings are invalid` — the default is never silently substituted |

## Local verification

```bash
go test -count=1 ./backend/safetyprofile ./backend/endpoints/application
```
