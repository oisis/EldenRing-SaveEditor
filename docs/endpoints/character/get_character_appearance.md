# GetCharacterAppearance

## Overview

`GetCharacterAppearance` returns the raw appearance stored in one physical
character slot of a save session that already exists in SaveEngine. It reads the
session's private snapshot only.

The session must have been created earlier by
[`LoadSave`](../savesession/load_save.md). `GetCharacterAppearance` never creates
one, so calling it before a successful `LoadSave` is an error, not an implicit
load. The endpoint opens no source file, returns no raw save byte, and modifies
nothing: neither the save, nor the session, nor any application state.

| | |
|---|---|
| EndpointID | `get_character_appearance` |
| Kind | Getter |
| Domain | `character` |
| Implementation status | implemented |
| Transport status | not exposed — there is no OpenAPI route in the local explorer (`backend/endpoints/swagger`), no Wails binding, no CLI command and no frontend. The endpoint is reachable from Go callers only. |
| Implementation source | [../../../backend/endpoints/character/get_character_appearance.go](../../../backend/endpoints/character/get_character_appearance.go) |
| Test source | [../../../backend/endpoints/character/get_character_appearance_test.go](../../../backend/endpoints/character/get_character_appearance_test.go) |
| Save access | read-only — the session's private in-memory snapshot; no file is opened |
| Mutation | none — the snapshot, the session, and the save file are left unchanged |

## Input

```go
func GetCharacterAppearance(engine *saveengine.Engine, saveSessionID string, characterID int) (GetCharacterAppearanceResult, error)
```

| Parameter | Type | Meaning |
|---|---|---|
| `engine` | `*saveengine.Engine` | The SaveEngine instance supplied by the backend caller. It owns the sessions; the endpoint never creates one. A `nil` engine is rejected. |
| `saveSessionID` | `string` | Identifier of an existing session, exactly as returned by `LoadSave`. It is passed to SaveEngine unchanged. |
| `characterID` | `int` | The physical slot index, `0` to `9`. It is the same index `GetSaveCharacters` reports positionally. |

### `saveSessionID`

- It is matched exactly and case-sensitively. It is never trimmed, normalised,
  or guessed, so `" <id>"`, `"<id> "`, and an upper-cased identifier are unknown
  values, not the session they resemble.
- Validation lives in SaveEngine. The endpoint holds no session-identifier rule
  of its own.

### `characterID`

- It is an index, not an identifier to search for: slot `n` is read directly.
- A value below `0` or above `9` is rejected. It is never clamped to the valid
  range and never resolved to a neighbouring slot.

## Output

```go
type GetCharacterAppearanceResult = saveengine.CharacterAppearance

type CharacterAppearance struct {
	SaveSessionID string    `json:"saveSessionID"`
	CharacterID   int       `json:"characterID"`
	Active        bool      `json:"active"`
	Gender        uint8     `json:"gender"`
	VoiceType     uint8     `json:"voiceType"`
	ModelIDs      [8]uint32 `json:"modelIDs"`
	FaceShape     [64]uint8 `json:"faceShape"`
	Body          [7]uint8  `json:"body"`
	Skin          [91]uint8 `json:"skin"`
}
```

| Field | Type | Meaning |
|---|---|---|
| `saveSessionID` | `string` | Identifier of the session that was read. It equals the requested value. |
| `characterID` | `int` | The requested slot index, `0` to `9`. It equals the requested value. |
| `active` | `bool` | `true` only when the slot's activity flag is exactly `1`. Any other flag value is not active. |
| `gender` | `uint8` | The stored body-type identifier of an active slot, as held in the player data. Always `0` for an inactive slot. |
| `voiceType` | `uint8` | The stored voice-type identifier of an active slot. Always `0` for an inactive slot. |
| `modelIDs` | `[8]uint32` | The eight stored model identifiers of an active slot, in save order: face, hair, eyes, eyebrows, beard, eyepatch, decal, eyelashes. Always all `0` for an inactive slot. |
| `faceShape` | `[64]uint8` | The 64 stored face-shape parameters of an active slot, in save order. Always all `0` for an inactive slot. |
| `body` | `[7]uint8` | The seven stored body proportions of an active slot, in save order: head, chest, abdomen, right arm, right leg, left arm, left leg. Always all `0` for an inactive slot. |
| `skin` | `[91]uint8` | The 91 stored skin and cosmetics parameters of an active slot, in save order. Always all `0` for an inactive slot. |

### Raw values only

Every field is the value stored in the save, reported exactly as read. The
endpoint computes nothing:

- No value is validated, normalised, clamped, repaired, or rejected. An unknown
  `gender`, an unknown `voiceType`, and a model identifier no in-game part uses
  are all returned as stored.
- Nothing is mapped to a name. No identifier is resolved against GameCatalog, no
  preset is matched, and no colour is converted.
- `modelIDs` keeps the full `uint32` each identifier is stored as. It is never
  narrowed to a byte, so a value above `255` survives unchanged.

### JSON shape

`faceShape`, `body`, and `skin` are fixed-size Go arrays, not byte slices, so
they serialise as JSON arrays of numbers — `[12, 240, 3, …]` — of exactly `64`,
`7`, and `91` elements. They are never encoded as hex, base64, or a string.
`modelIDs` serialises the same way, as exactly `8` numbers.

### What is not returned

The result contains only the fields above. It carries no character name, level,
class, play time, statistics, inventory, equipment, preset match, offsets, and no
raw bytes. In particular the opaque 64-byte block that sits between the face
shape and the body proportions is neither read nor reported. None of that is read
or computed to produce the result.

### Inactive and residual slots

An inactive slot is a normal result, not an error. It reports `active: false`
with `saveSessionID` and `characterID` filled in and every other field zeroed.

This holds for a residual slot too, where the raw appearance of a deleted
character is still present in the file: the activity flag alone decides what is
reported. An inactive slot's data is never searched and never read, so its
residual appearance is neither located nor decoded.

On any error the result is the zero value.

## Processing flow

1. The endpoint rejects a missing engine.
2. Everything else is delegated to SaveEngine, in this order: `saveSessionID` is
   validated, the session is looked up under the engine's own lock, and
   `characterID` is checked against the slot range.
3. SaveEngine reads the slot's activity flag through the codec's bounded,
   copying reads. An inactive slot returns immediately, without touching the
   slot data.
4. For an active slot only, SaveEngine locates the confirmed player-data anchor
   inside the data of that one slot and reads `gender` and `voiceType` backwards
   from it.
5. It then locates the first confirmed appearance block by its own header inside
   the same slot data, verifies the alignment and inner size that header
   declares, and decodes the model identifiers, face shape, body proportions, and
   skin parameters forwards from the block start. A healthy slot also carries
   later copies of the same header behind the sections this getter never touches;
   they are ignored, because the first block is the appearance the game reads.
6. The result is returned by value. The snapshot and the session model stay
   inside the package.

PC and PS4 differ in the base of the slot data only; the layout behind it is the
same, so both platforms run the same search and the same reads.

The endpoint is thin by design: it contains no SaveEngine rule, it holds no
knowledge of the save format, and there is no shared endpoint helper behind it.
It calls no other endpoint — in particular neither `LoadSave`, `GetLoadedSave`,
`CloseSave`, `GetSaveCharacters`, `GetCharacterProfile`, nor `GetCharacterStats`.

## Validation and errors

Every failure returns the zero result and changes nothing.

| Situation | Behaviour |
|---|---|
| `engine` is `nil` | `save engine is not available` — a backend wiring error, not client input. |
| `saveSessionID` is empty | `saveSessionID is required`. No lookup is attempted. |
| `saveSessionID` is unknown or closed | `unknown save session "<id>"`. A closed or never-created session is never resolved to a different one. |
| `characterID` is outside `0..9` | `characterID <id> is outside the range 0..9`. Checked only after the session resolves. |
| An active slot carries no player-data anchor | `character <id> carries no appearance player anchor`. `gender` and `voiceType` are addressed from that one confirmed anchor. |
| An active slot carries no appearance block | `character <id> carries no appearance block`. |
| The first appearance block does not fit inside the slot | `appearance block of character <id> does not fit into its slot`. |
| The first appearance block header declares an unexpected alignment or inner size | `appearance block of character <id> declares alignment <a> and inner size 0x<s>, want 4 and 0x120`. |
| A required range lies outside the snapshot | The read is rejected before it happens, and the error names the character slot involved. |

The last four rows are fail-closed by design: for an active slot the appearance
must be present and complete where the game keeps it. A missing block, an
unexpected header, and a block reaching past the end of the slot all fail; no
later block is tried as a fallback, and there is no default offset, no partial
result, and no guessed value.

An inactive or residual slot is not in this table: it is a successful result.

Stored values are never an error. No identifier, parameter, or colour is rejected
for being out of range, inconsistent, or implausible.

## Dependencies

- The endpoint delegates to `backend/saveengine` and calls no other endpoint.
- It reads no GameCatalog data. No field is looked up, named, or validated
  against the catalog.
- It does not import `backend/core`, `backend/db`, `backend/editor`,
  `backend/templates`, `backend/vm`, or `internal/`. SaveForge 2.0 is
  greenfield.

## Command-line verification

`GetCharacterAppearance` is verified through its tests; it has no transport. From
the repository root:

```bash
go test ./backend/saveengine -run '^TestGetCharacterAppearance' -count=1 -v
go test ./backend/endpoints/character -run '^TestGetCharacterAppearance' -count=1 -v
```

The tests build synthetic PC and PS4 containers inside `t.TempDir()`. They use no
real save file and no repository fixture, so they depend on nothing outside the
test process. The two platform fixtures place the anchor and the appearance block
at different positions inside the slot, so a fixed offset instead of a search
cannot pass both. The PC fixture carries a second, equally well-formed appearance
block with different values behind the first one, so a decoder reading anything
but the first block fails. They cover every field at its full length, a model
identifier above `255`, a residual slot whose raw appearance survives a cleared
flag, the rejected `characterID` values `-1` and `10`, and an active slot whose
appearance block is missing, truncated at the slot end, or declares an unexpected
inner size.

## Current limitations

- There is no transport. No OpenAPI route, no Wails binding, no CLI command, and
  no frontend reaches the endpoint.
- It reports the raw stored appearance only. No preset name is matched, no
  identifier is resolved to an in-game part, and no colour is converted.
- It is a getter. Changing the appearance is not possible: the session is
  read-only at this stage.
