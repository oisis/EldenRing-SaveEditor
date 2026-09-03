# SetSafetyProfile

## Overview

`SetSafetyProfile` stores the global Safety Profile of the host application and
returns the settings now in effect, in the same shape
[`GetSafetyProfile`](get_safety_profile.md) reports.

It is a `contract.Mutation` but **not a save-session mutation**: it advances no
`saveRevision`, touches no session, writes no save byte and produces no mutation
receipt. It writes exactly one host-local settings document, atomically.

| | |
|---|---|
| EndpointID | `set_safety_profile` |
| Kind | Mutation |
| Domain | `application` |
| Implementation status | implemented |
| Transport status | desktop bridge only — bound as a Wails method on `desktop.Bridge` and reached by the SaveForge frontend. It is deliberately **not** registered as an HTTP route of the local explorer and therefore does not appear in `tools/swagger/openapi.json` or in the Scalar portal: an operation the explorer cannot perform must never be advertised there. |
| Implementation source | [../../../backend/endpoints/application/set_safety_profile.go](../../../backend/endpoints/application/set_safety_profile.go) |
| Save access | none |
| Mutation | one host-local settings document, replaced atomically |

## Input

```go
func SetSafetyProfile(
	store *safetyprofile.Store,
	safetyProfile string,
) (SetSafetyProfileResult, error)
```

`safetyProfile` is passed through byte for byte. It is never trimmed, recased or
aliased, and it must be exactly one of `safe`, `expanded_limits` or `chaos`.

## Output

`SetSafetyProfileResult` is `SafetyProfileResult`: the profile now in effect,
the closed vocabulary and the product default. A client refreshes its cached
value from this answer rather than from the value it sent.

## Effect on other endpoints

Changing the profile changes what
[`GetItemDatabase`](../catalog/get_item_database.md),
[`GetOwnedItems`](../inventory/get_owned_items.md) and every batch item mutation
report and accept, because all of them resolve their decisions through the same
policy module. No save revision moves, so no `session.changed` event is
published; a client refreshes its catalog and owned-item views itself.

## Errors

| Condition | Result |
|---|---|
| the settings store is not wired | `application settings are not available` |
| `safetyProfile` is not one of the three known values | `unknown safety profile %q; expected "safe", "expanded_limits" or "chaos"` |
| the settings document cannot be written | the previous profile stays in effect in memory and on disk |

## Local verification

```bash
go test -count=1 ./backend/safetyprofile ./backend/endpoints/application
```
