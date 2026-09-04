# CheckForUpdates

## Overview

`CheckForUpdates` performs exactly one bounded HTTPS request against the
project's official GitHub Releases and reports whether a newer stable release
exists.

It runs only on an explicit user action. There is no background schedule, no
retry and no automatic check on start-up, and the endpoint downloads and
installs nothing. Drafts and prereleases are discarded, so an unreleased tag can
never be reported as available.

The request is anonymous by construction: it carries an `Accept` and a
`User-Agent` header and nothing else — no token, no cookie and no user identity
— and it uses its own HTTP client rather than the process default, so it cannot
inherit a cookie jar or redirect policy installed elsewhere. The standard Go
transport may still honor the host's proxy configuration.

| | |
|---|---|
| EndpointID | `check_for_updates` |
| Kind | Getter |
| Domain | `application` |
| Implementation status | implemented |
| Transport status | desktop bridge only — a Wails method of `desktop.Bridge`; deliberately not an HTTP route of the local explorer and therefore absent from OpenAPI and Scalar |
| Implementation source | [../../../backend/endpoints/application](../../../backend/endpoints/application) |
| Save access | none |

## Input

```go
func CheckForUpdates(
	ctx context.Context,
	applicationVersion string,
	endpointOverride string,
) (CheckForUpdatesResult, error)
```

## Output

```go
type CheckForUpdatesResult struct {
	Status             string `json:"status"`
	CurrentVersion     string `json:"currentVersion"`
	LatestVersion      string `json:"latestVersion,omitempty"`
	ReleaseURL         string `json:"releaseURL,omitempty"`
	PublishedAt        string `json:"publishedAt,omitempty"`
	ComparisonPossible bool   `json:"comparisonPossible"`
}
```

`status` is one of four stable codes; the frontend owns the wording:

| Status | Meaning |
|---|---|
| `current` | the running version is at least the newest stable release |
| `available` | a newer stable release exists |
| `unknown` | the running version carries no comparable number, such as a development build |
| `unavailable` | the project has published no stable release yet |

`endpointOverride` exists for a test against a local mock server and is empty in
production. The bridge never forwards a value for it, so a frontend cannot
redirect the check.

The request is bounded twice: a ten-second timeout covering connection and body,
and a one-megabyte limit checked with a one-byte overflow probe. Redirects are
accepted only within the original scheme and host.

## Errors

| Condition | Result |
|---|---|
| `applicationVersion` is empty | `application version is required` |
| the service cannot be reached | `the update check could not reach the release service` |
| the service answers with a non-200 status | `the release service did not answer the update check` |
| the service redirects outside the approved origin | `the update check could not reach the release service` |
| the answer exceeds one megabyte | `the update check answer was too large` |
| the answer cannot be read or understood | `the update check answer could not be read` / `… could not be understood` |

The upstream answer is never repeated into the failure. A rate-limit message
carries an address and reads as an application fault, and neither is actionable
for the user.

## Local verification

```bash
go test -count=1 ./backend/hostsettings ./backend/endpoints/application
```
