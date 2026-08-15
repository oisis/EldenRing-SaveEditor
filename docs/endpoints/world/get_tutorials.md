# GetTutorials

## Overview

`GetTutorials` returns the 72 user-facing tutorial records that have an official
non-empty English title, together with their current unlock state for one
character slot. It is a read-only join between `TutorialDocument` resources in
GameCatalog and the raw `TutorialData` membership read by SaveEngine.

| | |
|---|---|
| EndpointID | `get_tutorials` |
| Kind | Getter |
| Domain | `world` |
| Implementation status | implemented |
| Transport | `GET /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/tutorials`, loopback explorer only |
| Save access | read-only private session snapshot |

## Input

```go
func GetTutorials(
    engine *saveengine.Engine,
    gameCatalog *gamecatalog.Catalog,
    saveSessionID string,
    characterID int,
    availabilityFilter string,
) (GetTutorialsResult, error)
```

`availabilityFilter` is matched exactly:

- `""` returns every declared tutorial;
- `"unlocked"` returns only IDs present in `TutorialData`;
- `"locked"` returns only IDs absent from `TutorialData`.

No whitespace trimming, case folding or alias such as `all` is applied.

## Catalog contract

The generator reads `TutorialParam.csv` and the official English
`TutorialTitle` FMG files. A public tutorial resource requires:

- kind `tutorial`;
- a decimal key equal to its `TutorialParam` row ID;
- a known, non-zero tutorial ID;
- a known, non-empty official title.

Rows `0`, `1` and `2` are technical/dummy records and are excluded. Eleven
additional gameplay rows have no official title and are short control prompts;
they are preserved in the source data but are not converted into guessed public
resources. The resulting catalog contains 72 tutorials.

The raw tutorial ID and both provenance records remain available through
[`GetResource`](../catalog/get_resource.md). The world getter represents that ID
once as its decimal resource key and does not duplicate it in a numeric field.

## Save contract

`TutorialData` is a dynamic block behind `GaItemGameData`:

| Offset | Value |
|---|---|
| `+0x00` | two uninterpreted `uint16` values |
| `+0x04` | declared payload size (`uint32`) |
| `+0x08` | tutorial count (`uint32`) |
| `+0x0C` | `count` little-endian `uint32` TutorialParam row IDs |

The reader always uses the declared payload size; it does not assume the legacy
default `0x400`. It rejects a count outside the payload capacity, a count above
the confirmed hard cap of 255, an incomplete header and any range outside the
character slot or snapshot.

The physical format and membership semantics are identical in SaveForge v1.5.8
and v1.6.8. The verified native transition in the legacy research appended row
ID `2010` after the Item Crafting tutorial was triggered.

An inactive or residual slot returns `active=false`. Its slot data is never
searched, and every declared tutorial is treated as locked before applying the
filter.

## Result

```go
type GetTutorialsResult struct {
    SaveSessionID string          `json:"saveSessionID"`
    CharacterID   int             `json:"characterID"`
    Active        bool            `json:"active"`
    Tutorials     []TutorialEntry `json:"tutorials"`
}

type TutorialEntry struct {
    Kind     schema.ResourceKind `json:"kind"`
    Key      string              `json:"key"`
    Title    string              `json:"title"`
    Unlocked bool                `json:"unlocked"`
}
```

`tutorials` is non-nil and ordered by title, then key. Filtering never changes
that order.

## Errors

The getter fails without a partial result for a nil dependency, invalid filter,
unknown session, character outside `0..9`, invalid catalog resource, missing
anchor, malformed dynamic length, invalid tutorial count or truncated snapshot.

## Related endpoint

`SetTutorialUnlocked` remains contract-only. It has no runtime handler, HTTP
route, OpenAPI operation or Scalar page.
