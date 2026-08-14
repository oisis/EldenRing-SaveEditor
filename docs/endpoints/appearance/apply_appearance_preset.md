# ApplyAppearancePreset

## Overview

`ApplyAppearancePreset` resolves one stored appearance preset and applies its
complete appearance through the existing `SetCharacterAppearance` SaveEngine
operation. It introduces no second binary writer or parallel mutation rule.

| | |
|---|---|
| EndpointID | `apply_appearance_preset` |
| Kind | Mutation |
| Domain | `appearance` |
| Implementation status | implemented |
| Transport status | transport-exposed — `PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/appearance/preset` of the local OpenAPI explorer, registered only without `-allow-external-bind` |
| Implementation source | [../../../backend/endpoints/appearance/apply_appearance_preset.go](../../../backend/endpoints/appearance/apply_appearance_preset.go) |
| Test source | [../../../backend/endpoints/appearance/apply_appearance_preset_test.go](../../../backend/endpoints/appearance/apply_appearance_preset_test.go) |
| GameCatalog access | one exact preset from `presets/appearance.json` and the confirmed Type A/Type B model mapping |
| Save access | exactly one call to `SaveEngine.SetCharacterAppearance`; persistence remains a separate `WriteSave` operation |

## Input

```go
func ApplyAppearancePreset(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
	presetID string,
	expectedRevision string,
) (ApplyAppearancePresetResult, error)
```

The HTTP body is:

```json
{
  "presetID": "yennefer-sorceress-from-the-witcher",
  "expectedRevision": "0"
}
```

`presetID` is required and matched exactly and case-sensitively. It is never
trimmed, normalised, aliased or replaced with another preset.

`expectedRevision` must be the canonical decimal revision currently held by
the session. The transport rejects unknown JSON fields.

## Processing

The endpoint performs four steps:

1. Read an independent copy of the appearance presets from GameCatalog.
2. Resolve exactly one `presetID`.
3. Convert the preset's eight UI-facing model selections to confirmed raw
   PartsIds. Type A hair uses its confirmed non-sequential map; Type B uses the
   finite confirmed maps for each model field. An absent mapping is an error,
   never a guessed value.
4. Pass the complete gender, voice type, raw model IDs, face shape, body and
   skin model plus `expectedRevision` to `SaveEngine.SetCharacterAppearance`.

SaveEngine remains the single owner of validation, atomicity, revision handling,
rollback and the PC/PS4 DFLT/ZSTD write paths. The endpoint never merges a
preset with the current save and never applies only selected fields.

The model mappings reproduce the behavior shared by SaveForge 1.5.8 and 1.6.8,
but the 2.0 implementation fails closed where legacy Type A hair code had a
one-based fallback for an unknown value.

## Result

```go
type ApplyAppearancePresetResult struct {
	SaveSessionID string                               `json:"saveSessionID"`
	SaveRevision  string                               `json:"saveRevision"`
	CharacterID   int                                  `json:"characterID"`
	PresetID      string                               `json:"presetID"`
	Appearance    saveengine.CharacterAppearanceValues `json:"appearance"`
}
```

The receipt identifies the selected preset, echoes the complete committed
appearance and returns the revision required by the next mutation.

## Errors and atomicity

The whole call fails when the engine or GameCatalog is unavailable, `presetID`
is empty or unknown, a stored model selection has no confirmed mapping, the
revision is invalid or stale, the character is inactive, or the shared complete
appearance mutation cannot safely locate, write or verify the fields.

Every failure returns an empty result and leaves the snapshot, revision and
dirty flag unchanged. There is no fallback preset, partial application or
best-effort model conversion.

## Transport

```text
PUT /api/v1/save-sessions/{saveSessionID}/characters/{characterID}/appearance/preset
```

The route is available only in the explorer's local loopback mode. Success
returns `200`; request and endpoint errors return `400`. There is no Wails,
frontend or permanent CLI binding.

## Local verification

```text
go test ./backend/gamecatalog -run 'TestAppearanceModelIDs' -count=1
go test ./backend/endpoints/appearance -run 'TestApplyAppearancePreset' -count=1
go test ./tools/swagger -run 'TestApplyAppearancePresetRoute' -count=1
```

The endpoint tests cover stored Type A and Type B presets, exact identifier
matching and fail-closed unconfirmed mappings. Binary platform, compression,
rollback and reload behavior remain covered by the dedicated
`SetCharacterAppearance` SaveEngine tests and are not duplicated here.
