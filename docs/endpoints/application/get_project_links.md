# GetProjectLinks

## Overview

`GetProjectLinks` returns the closed set of approved project addresses the
`About & Updates` screen may open, in the order it presents them.

The table is a compile-time constant. Nothing at runtime can add an entry, and
no configuration, save or frontend argument feeds it. It exists so the frontend
can ask the host to open a destination by identifier: there is deliberately no
bridge method that opens an arbitrary address the frontend supplies.

| | |
|---|---|
| EndpointID | `get_project_links` |
| Kind | Getter |
| Domain | `application` |
| Implementation status | implemented |
| Transport status | desktop bridge only — a Wails method of `desktop.Bridge`; deliberately not an HTTP route of the local explorer and therefore absent from OpenAPI and Scalar |
| Implementation source | [../../../backend/endpoints/application](../../../backend/endpoints/application) |
| Save access | none |

## Input

```go
func GetProjectLinks() (GetProjectLinksResult, error)
```

## Output

```go
type ProjectLink struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type GetProjectLinksResult struct {
	Links []ProjectLink `json:"links"`
}
```

The four approved identifiers are `repository`, `releases`, `sponsor_coffee` and
`sponsor_bitcoin`.

## Errors

The getter cannot fail. The companion resolver used by the bridge does:

| Condition | Result |
|---|---|
| `ResolveProjectLink` receives an identifier that is not in the table | `unknown project link …` — no address is produced |

## Local verification

```bash
go test -count=1 ./backend/hostsettings ./backend/endpoints/application
```
