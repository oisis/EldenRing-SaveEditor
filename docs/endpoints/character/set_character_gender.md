# SetCharacterGender

## Overview

`SetCharacterGender` switches one active character to body type `0` (Type B)
or `1` (Type A). A body-type change is a complete appearance operation: the
endpoint applies the confirmed Ciri default for Type B or Geralt default for
Type A through `SaveEngine.SetCharacterGenderAppearance`, the gender entry
point of the one private appearance writer every appearance mutation shares.

| | |
|---|---|
| EndpointID | `set_character_gender` |
| Kind | Mutation |
| Domain | `character` |
| Implementation status | implemented |
| Transport status | transport-exposed — `PATCH /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/gender` of the local OpenAPI explorer, registered only without `-allow-external-bind` |
| Implementation source | [../../../backend/endpoints/character/set_character_gender.go](../../../backend/endpoints/character/set_character_gender.go) |
| Test source | [../../../backend/endpoints/character/set_character_gender_test.go](../../../backend/endpoints/character/set_character_gender_test.go) |
| GameCatalog access | the confirmed default appearance preset for the requested body type |
| Save access | exactly one call to `SaveEngine.SetCharacterGenderAppearance`; persistence remains a separate `WriteSave` operation |

## Input

```go
func SetCharacterGender(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	gender uint8,
	expectedRevision string,
) (SetCharacterGenderResult, error)
```

The HTTP body is:

```json
{
  "gender": 0,
  "expectedRevision": "0"
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
receipt members and `characterID`, `presetID`, `appearance` all sit at the top
level, and there is no nested `receipt` object.

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
  `set_character_gender`.
- `changedScopes` are exactly `save.session`, `character.profile`,
  `character.appearance`, `diagnostics.report`, in that canonical order.

`SetCharacterAppearance`, `SetCharacterGender` and `ApplyAppearancePreset` share
one private SaveEngine appearance writer, and the writer receives its operation
kind from the public entry point. Each of the three therefore always reports its
own kind and never one of the other two.

`gender` is the raw body-type value used by the save:

- `0` selects Type B and applies
  `ciri-the-princess-of-cintra-from-witcher`;
- `1` selects Type A and applies `geralt-of-rivia-the-witcher`.

Any other value is rejected. The transport uses a nullable internal field so
an omitted `gender` is not confused with the valid value zero.

`expectedRevision` must be the canonical decimal revision currently held by
the session. The transport rejects unknown JSON fields.

## Processing

The endpoint performs four steps:

1. Resolve the confirmed default preset for the requested body type from the
   loaded GameCatalog.
2. Resolve its eight UI-facing model selections through the same confirmed
   Type A/Type B PartsId mapping used by `ApplyAppearancePreset`, including the
   shared Face / Bone Structure table where UI 1-6 map to PartsIds 0 to 50.
3. Build the complete appearance assignment: gender, voice type, model IDs,
   face shape, body and skin.
4. Pass that assignment and `expectedRevision` to
   `SaveEngine.SetCharacterGenderAppearance`.

The operation intentionally does not preserve the previous face or voice. This
matches the behavior shared by SaveForge 1.5.8 and 1.6.10, where changing body
type applies the corresponding complete default preset.

SaveEngine owns validation, atomicity, revision handling, rollback, FACE layout
and the PC/PS4 DFLT/ZSTD paths. The endpoint owns no binary offset or writer.

## Result

```go
type SetCharacterGenderResult struct {
	saveengine.MutationReceipt
	CharacterID   int                                  `json:"characterID"`
	PresetID      string                               `json:"presetID"`
	Appearance    saveengine.CharacterAppearanceValues `json:"appearance"`
}
```

The result identifies the default preset, echoes the complete committed
appearance and returns the revision required by the next mutation.

## Errors and atomicity

The whole call fails when the engine or GameCatalog is unavailable, `gender`
is outside `0..1`, the corresponding default preset is missing, its model
mapping is unsupported, the revision is invalid or stale, the character is
inactive, or the shared complete appearance mutation cannot safely locate,
write or verify its fields.

Every failure returns an empty result and leaves the snapshot, revision and
dirty flag unchanged. There is no fallback preset, partial gender-only write or
best-effort conversion.

## Transport

```text
PATCH /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/gender
```

The route is available only in the explorer's local loopback mode. Success
returns `200`; request and endpoint errors return `400`. There is no Wails,
frontend or permanent CLI binding.

## Local verification

```text
go test ./backend/gamecatalog -run 'TestCatalog.*AppearancePreset' -count=1
go test ./backend/endpoints/character -run 'TestSetCharacterGender' -count=1
go test ./tools/swagger -run 'TestSetCharacterGenderRoute' -count=1
```

The endpoint test covers both confirmed defaults and invalid input. Binary
platform, compression, rollback and reload behavior remains covered by the
dedicated `SetCharacterAppearance` SaveEngine tests and is not duplicated here.
