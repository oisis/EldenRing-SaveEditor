# ApplyFavoritePreset

## Overview

`ApplyFavoritePreset` applies the confirmed appearance fields from the
specified Mirror Favorites preset slot stored in `UserData10` to an active
character of an existing save session under `expectedRevision` control. It
operates on the session's private snapshot only.

The Mirror Favorites slots are shared across all 10 character slots of the save
file (they live globally in `UserData10`). Applying a preset copies the
appearance model into the character slot, while the preset in `UserData10`
remains unchanged.

The session must have been created earlier by
[`LoadSave`](../savesession/load_save.md). `ApplyFavoritePreset` opens no source
file, reads no GameCatalog, requires no selection object, and persists nothing
directly; persistence is owned by a separate
[`WriteSave`](../savesession/write_save.md).

| | |
|---|---|
| EndpointID | `apply_favorite_preset` |
| Kind | Mutation |
| Domain | `favorites` |
| Implementation status | implemented |
| Transport status | transport-exposed — `PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/appearance/favorite-preset` of the local OpenAPI explorer (`tools/swagger`), registered only when the explorer runs without `-allow-external-bind`. There is no Wails binding, no frontend view and no CLI command. |
| Implementation source | [../../../backend/endpoints/favorites/apply_favorite_preset.go](../../../backend/endpoints/favorites/apply_favorite_preset.go) |
| Test source | [../../../backend/endpoints/favorites/apply_favorite_preset_test.go](../../../backend/endpoints/favorites/apply_favorite_preset_test.go) |
| Data source | the global Mirror Favorites slot in `UserData10` and the target character's appearance in slot data, mutated by SaveEngine |
| Save access | private session snapshot mutation with verification and rollback; no file is opened |
| Mutation | writes the confirmed appearance fields to the target character slot; advances `saveRevision`, sets `dirty` flag and registers a standard single undo point for the character |

## Input

```go
func ApplyFavoritePreset(
	engine *saveengine.Engine,
	saveSessionID string,
	characterID int,
	favoriteSlotID int,
	expectedRevision string,
) (ApplyFavoritePresetResult, error)
```

| Parameter | Type | Meaning |
|---|---|---|
| `engine` | `*saveengine.Engine` | The SaveEngine instance that owns the loaded sessions. A `nil` engine is rejected. |
| `saveSessionID` | `string` | Identifier of an already loaded session. It is passed to SaveEngine unchanged. |
| `characterID` | `int` | Slot index in `0..9` identifying the active target character. |
| `favoriteSlotID` | `int` | Slot index in `0..14` identifying which of the 15 Mirror Favorites preset slots to apply. |
| `expectedRevision` | `string` | Canonical decimal `saveRevision` matching the current revision of the session. |

### `saveSessionID`

- Matched exactly and case-sensitively by SaveEngine. It is never trimmed,
  normalised or guessed, so an empty, unknown or already closed identifier is
  rejected before any mutation occurs.

### `characterID`

- An integer in `0..9`.
- Must point to an active character slot (`UserData10` active flag equals `1`).
- The target character's appearance anchor and `FACE` block must be well-formed.

### `favoriteSlotID`

- An integer in `0..14`.
- Must point to an active, populated Mirror Favorites preset slot (`"FACE"` magic at `+0x18`).
- Any value outside `0..14` (e.g. `-1` or `15`) is rejected.
- The preset slot must have a valid body type (`0` or `1`).

### `expectedRevision`

- Must be the canonical decimal string representation of a non-negative integer
  without leading zeros (e.g. `"0"`, `"1"`).
- Checked under the session lock inside `commitCharacterRevision`. If it does not match
  the session's current `saveRevision`, the operation fails with no changes.

## Result

```go
type ApplyFavoritePresetResult struct {
	MutationReceipt
	CharacterID    int    `json:"characterID"`
	FavoriteSlotID int    `json:"favoriteSlotID"`
}

type MutationReceipt struct {
	OperationID   string   `json:"operationID"`
	OperationKind string   `json:"operationKind"`
	SaveSessionID string   `json:"saveSessionID"`
	SaveRevision  string   `json:"saveRevision"`
	ChangedScopes []string `json:"changedScopes"`
}
```

The receipt is embedded anonymously, so the JSON result is flat: the five
receipt members and `characterID`, `favoriteSlotID` all sit at the top level,
and there is no nested `receipt` object.

The embedded `saveengine.MutationReceipt` is exactly the receipt the central
SaveEngine commit path produced for this execution. Nothing here is
reassembled from the EndpointID, the session, the revision or a scope lookup.

- `operationID` names this one execution. It is opaque and unpredictable.
  Identifiers do not repeat among the receipts issued by one running SaveEngine
  instance. That guarantee does not currently cover application restarts:
  uniqueness across restarts requires a persistent operation journal and stays
  outside this stage. A rejected call returns the complete zero result and no
  `operationID` at all.
- `operationKind` is the stable kind of the mutation and is always exactly
  `apply_favorite_preset`.
- `changedScopes` are exactly `save.session`, `character.profile`,
  `character.appearance`, `diagnostics.report`, in that canonical order.

This endpoint writes character appearance rather than a preset slot, so its
changed scopes are the appearance scopes and not the `favorites` scope of
`SetFavoritePreset` and `DeleteFavoritePreset`.

| Field | Type | Meaning |
|---|---|---|
| `operationID` | `string` | Opaque identifier of this one execution. |
| `operationKind` | `string` | Stable kind of the mutation, exactly `apply_favorite_preset`. |
| `saveSessionID` | `string` | Identifier of the session that was modified. |
| `saveRevision` | `string` | New canonical decimal save revision after the mutation (incremented by 1). |
| `changedScopes` | `[]string` | Backend read scopes this mutation invalidated, in the one canonical order. |
| `characterID` | `int` | Target character slot index in `0..9` that was modified. |
| `favoriteSlotID` | `int` | Mirror Favorites preset slot index in `0..14` that was applied. |

## Applied and preserved fields

The preset transfers the following fields from the Mirror Favorites slot to the character:

- **Model IDs**: 8 uint32 values (face, hair, eye, eyebrow, beard, eyepatch, decal, eyelash);
- **Face shape sliders**: 64 bytes;
- **Body proportions**: 7 bytes (head, chest, abdomen, right arm, right leg, left arm, left leg);
- **Skin and cosmetics**: 91 bytes;
- **Gender**: mapped from the inverted preset body type: preset `0` (male) → character Gender `1`, preset `1` (female) → character Gender `0`.

The following character fields remain **untouched**:

- **VoiceType**: not represented in Mirror Favorites; preserved unchanged;
- **Opaque FaceData block** (`0x70..0xB0`): preserved unchanged;
- **Equipment and inventory**: unchanged;
- **Mirror Favorites preset in UserData10**: unchanged.

The two confirmed dependent sex-flag bytes at `0x125` and `0x126` in the character's `FACE` block are reset to zero as in `SetCharacterAppearance`.

## Verification and error handling

- If `favoriteSlotID` is outside `0..14` or the slot lies outside `UserData10` bounds, the mutation is rejected fail-closed before any byte is written.
- If the preset slot is empty or not active (missing `"FACE"` magic at `+0x18`), the mutation fails without changes.
- If the preset slot carries an invalid body type (outside `0..1`), the mutation is rejected without changes.
- If `characterID` is outside `0..9` or the target character is inactive, the mutation is rejected immediately.
- If `expectedRevision` does not match the session's current revision, the operation fails before mutation.
- Writing to character appearance is backed by the shared private appearance writer with rollback on write failure.
- An idempotent apply (applying identical appearance data) advances revision as normal, but records no empty undo point.

## Local verification

```bash
go test ./backend/saveengine -run '^TestApplyFavoritePreset' -count=1
go test ./backend/endpoints/favorites -run '^TestApplyFavoritePreset' -count=1
go test ./tools/swagger -run '^TestApplyFavoritePreset' -count=1
```
