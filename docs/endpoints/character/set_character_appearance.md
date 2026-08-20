# SetCharacterAppearance

## Overview

`SetCharacterAppearance` atomically replaces the complete raw appearance model
of one active character. It changes the session's private snapshot only; the
source save remains untouched until
[`WriteSave`](../savesession/write_save.md) is called.

| | |
|---|---|
| EndpointID | `set_character_appearance` |
| Kind | Mutation |
| Domain | `character` |
| Implementation status | implemented |
| Transport status | transport-exposed — `PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/appearance` of the loopback-only local explorer |
| Implementation source | [../../../backend/endpoints/character/set_character_appearance.go](../../../backend/endpoints/character/set_character_appearance.go) |
| Endpoint tests | [../../../backend/endpoints/character/set_character_appearance_test.go](../../../backend/endpoints/character/set_character_appearance_test.go) |
| SaveEngine source | [../../../backend/saveengine/set_character_appearance.go](../../../backend/saveengine/set_character_appearance.go) |
| SaveEngine tests | [../../../backend/saveengine/set_character_appearance_test.go](../../../backend/saveengine/set_character_appearance_test.go) |
| Mutation | complete raw appearance assignment; advances `saveRevision` by 1 |

## Input

```go
func SetCharacterAppearance(
	engine *saveengine.Engine,
	saveSessionID string,
	characterID int,
	appearance CharacterAppearanceValues,
	expectedRevision string,
) (SetCharacterAppearanceResult, error)

type CharacterAppearanceValues struct {
	Gender    uint8     `json:"gender"`
	VoiceType uint8     `json:"voiceType"`
	ModelIDs  [8]uint32 `json:"modelIDs"`
	FaceShape [64]uint8 `json:"faceShape"`
	Body      [7]uint8  `json:"body"`
	Skin      [91]uint8 `json:"skin"`
}
```

`saveSessionID` identifies an existing session exactly, `characterID` is a
physical slot from `0` through `9`, and `expectedRevision` must be the current
canonical decimal revision. The slot must be active.

The complete appearance is required. `gender` accepts `0` or `1`, and
`voiceType` accepts `0` through `5`. The four arrays contain exactly `8`, `64`,
`7`, and `91` values. Their fixed Go types enforce the physical `uint32` and
byte ranges. Model identifiers are raw save values and are not narrowed,
clamped, named, or resolved against GameCatalog.

The HTTP body nests this model under `appearance` and adds
`expectedRevision`. It requires `application/json`, rejects unknown fields,
omitted scalar values, and every array with a wrong length.

## Output

```go
type SetCharacterAppearanceResult struct {
	SaveSessionID string                    `json:"saveSessionID"`
	SaveRevision  string                    `json:"saveRevision"`
	CharacterID   int                       `json:"characterID"`
	Appearance    CharacterAppearanceValues `json:"appearance"`
}
```

The receipt returns the complete accepted model and the revision created by the
mutation. It contains no offsets, preset identity, names, or private save bytes.

## Save mutation

SaveEngine reuses the same bounded anchor and first-FACE-block locator as
[`GetCharacterAppearance`](get_character_appearance.md). It writes:

- the one-byte gender and voice-type fields in PlayerGameData;
- eight little-endian `uint32` model identifiers at FACE offsets
  `0x10..0x2F`;
- the face-shape, body, and skin byte blocks at FACE offsets `0x30`, `0xB0`,
  and `0xB7`;
- zero to the two confirmed dependent sex-flag bytes at FACE offsets `0x125`
  and `0x126`.

The FACE header, the opaque block at `0x70..0xAF`, all other unknown bytes, and
every later FACE block remain unchanged. PC and PS4 share this layout behind
their existing platform-specific slot bases.

## Atomicity and failure behavior

Validation, location, mutation, verification, and rollback run under the
SaveEngine mutex. Every original field is read before the first write. A write
or verification failure restores gender, voice type, and the complete original
FACE block; revision and dirty state then remain unchanged.

A successful request advances `saveRevision` by exactly one and marks the
session dirty, including an idempotent assignment. The endpoint rejects a
missing engine, invalid or stale revision, invalid character index, inactive
slot, invalid gender or voice type, malformed HTTP model, missing anchor,
missing or invalid FACE block, truncated range, or failed write. It never falls
back to a later block or a guessed offset.

## Dependencies and scope

- The endpoint delegates one mutation to `backend/saveengine` and calls no
  other endpoint.
- It reads no GameCatalog data. Preset resolution and UI model mapping belong
  to the separate `ApplyAppearancePreset` contract.
- It creates no runtime or build dependency on either legacy SaveForge tree.

The confirmed layout and dependent sex-flag reset agree between SaveForge
1.5.8 and 1.6.10. Synthetic PC and PS4 tests verify exact byte preservation and
`WriteSave`/`LoadSave` persistence without modifying a real save.
