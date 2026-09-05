# GetHostSettings

## Overview

`GetHostSettings` returns the persistent host application settings that are not
save data: the two Save behavior preferences of `Tools → Settings`.

They belong to no save session, never enter a snapshot or a recovery journal,
and survive closing a save. The result also states whether this host has a
configuration directory and whether it has a local log directory. The log
directory is owned by the diagnostic service, not by the settings store: this
endpoint only reports whether one exists so the interface can enable or disable
`Open log directory`. Debug Mode itself is deliberately absent here — it is
runtime state with a single owner and is read through `GetDiagnosticMode`.

| | |
|---|---|
| EndpointID | `get_host_settings` |
| Kind | Getter |
| Domain | `application` |
| Implementation status | implemented |
| Transport status | desktop bridge only — a Wails method of `desktop.Bridge`; deliberately not an HTTP route of the local explorer and therefore absent from OpenAPI and Scalar |
| Implementation source | [../../../backend/endpoints/application](../../../backend/endpoints/application) |
| Save access | none |

## Input

```go
func GetHostSettings(
	store *hostsettings.Store, diagnosticService *diagnostics.Service,
) (HostSettingsResult, error)
```

## Output

```go
type HostSettingsResult struct {
	SkipReviewForNormalRisk       bool     `json:"skipReviewForNormalRisk"`
	RemoteBackupPolicy            string   `json:"remoteBackupPolicy"`
	AvailableRemoteBackupPolicies []string `json:"availableRemoteBackupPolicies"`
	DefaultRemoteBackupPolicy     string   `json:"defaultRemoteBackupPolicy"`
	ConfigurationDirectoryExists  bool     `json:"configurationDirectoryExists"`
	LogDirectoryExists            bool     `json:"logDirectoryExists"`
}
```

`availableRemoteBackupPolicies` is the closed vocabulary in its canonical order:
`ask` and `always`. There is deliberately no third value: a target save that
already exists is always backed up before it is replaced, so a "never back up"
mode cannot be expressed at all.

The directory flags are booleans rather than paths. The frontend never
receives a host path from this endpoint and never sends one back; opening a
directory is reached through the bridge by identifier instead.

## Errors

| Condition | Result |
|---|---|
| the settings store is not wired | `host settings store is required` |
| the stored settings document is unreadable or carries an unknown policy | `host settings are invalid` — the defaults are never silently substituted |
