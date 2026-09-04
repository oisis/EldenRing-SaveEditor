# ImportBuildTemplate

## Overview

`ImportBuildTemplate` adds a Build Template document the user picked in the
native file dialog to the local templates library.

The import is local only. There is deliberately no way to state a URL and
nothing here performs a network request. The chosen file is read under a size
bound and validated by the library's own `DecodeTemplate`, which rejects unknown
fields, trailing data and every schema violation; the import adds no second,
weaker check of its own.

Cancelling the dialog never reaches this endpoint: the bridge treats the empty
path as an ordinary outcome and stores nothing.

| | |
|---|---|
| EndpointID | `import_build_template` |
| Kind | Mutation |
| Domain | `templates` |
| Implementation status | implemented |
| Transport status | desktop bridge only — a Wails method of `desktop.Bridge`; deliberately not an HTTP route of the local explorer and therefore absent from OpenAPI and Scalar |
| Implementation source | [../../../backend/endpoints/templates](../../../backend/endpoints/templates) |
| Save access | none |

## Input

```go
func ImportBuildTemplate(
	store *buildtemplates.Store,
	source string,
) (ImportBuildTemplateResult, error)
```

## Output

```go
type ImportBuildTemplateResult struct {
	TemplateID       string `json:"templateID"`
	TemplateRevision string `json:"templateRevision"`
}
```

## Errors

| Condition | Result |
|---|---|
| the templates store is not wired | `templates store is not available` |
| `source` is empty | `a template import needs a source document` |
| the file cannot be opened or read | `the selected template document could not be read` — the host path is not repeated back |
| the file is not a regular file | `the selected template document is not a regular file` |
| the file exceeds 8 MiB | `a Build Template document must not exceed …` |
| the document fails schema validation | `the selected document is not a valid Build Template: …` |

Every refusal stores nothing: the library is written only after validation
succeeds.

## Local verification

```bash
go test -count=1 ./backend/buildtemplates ./backend/endpoints/templates
```
