# GetAppearancePresets

## Overview

`GetAppearancePresets` reports the appearance presets the backend offers today,
reduced to the metadata a preset list needs.

The presets live in
[`backend/gamecatalog/data/presets/appearance.json`](../../../backend/gamecatalog/data/presets/appearance.json)
and their preview images live in
[`backend/gamecatalog/data/assets/appearance`](../../../backend/gamecatalog/data/assets/appearance).
That file is the authoritative source of the preset data for SaveForge 2.0 and
for this endpoint. `GameCatalog` loads and validates it once, when it is built,
including the existence of every referenced asset, and the endpoint only reads
the already loaded presets: it opens no file per call.

| | |
|---|---|
| EndpointID | `get_appearance_presets` |
| Kind | Getter |
| Domain | `appearance` |
| Implementation status | implemented |
| Transport status | transport-exposed — `GET /api/v1/appearance/presets` of the local OpenAPI explorer (`backend/swagger`). No Wails binding and no permanent CLI command reach it. |
| Implementation source | [../../../backend/endpoints/appearance/get_appearance_presets.go](../../../backend/endpoints/appearance/get_appearance_presets.go) |
| Test source | [../../../backend/endpoints/appearance/get_appearance_presets_test.go](../../../backend/endpoints/appearance/get_appearance_presets_test.go) |
| Data source | [`backend/gamecatalog/data/presets/appearance.json`](../../../backend/gamecatalog/data/presets/appearance.json), loaded and validated by `GameCatalog` |
| Asset directory | [`backend/gamecatalog/data/assets/appearance`](../../../backend/gamecatalog/data/assets/appearance) — 20 `.jpg` files, one per preset |
| Save access | none — the endpoint never opens, reads, or writes a save |
| Mutation | none — the endpoint builds a result and modifies nothing |

## Input

The endpoint takes the loaded catalog and exactly two public parameters:

```go
func GetAppearancePresets(
	gameCatalog *gamecatalog.Catalog,
	search string,
	tags []string,
) (GetAppearancePresetsResult, error)
```

| Parameter | Type | Meaning |
|---|---|---|
| `gameCatalog` | `*gamecatalog.Catalog` | The already loaded catalog, supplied by the backend caller. It owns the preset data; the endpoint never loads it. |
| `search` | `string` | Case-insensitive substring filter over `id` and `name`. Empty means no filtering. |
| `tags` | `[]string` | Tag filter with AND semantics. `nil` or empty means no filtering. |

Matching rules:

- `search` is lowercased on both sides and matched as a substring against the
  preset `id` and the preset `name`. A preset matching either one is returned.
- `search` is never trimmed and never normalised. `" geralt"` therefore matches
  nothing, because no identifier and no name contains a leading space.
- Every tag in `tags` must be present on a preset for it to be returned, so the
  filter uses AND semantics, not OR.
- Tags are compared exactly and case-sensitively. `"Witcher"` and `" witcher"`
  are different tags, not the tag `witcher`.
- An empty tag is rejected instead of being ignored.
- `search` and `tags` are combined: a preset must satisfy both filters.

The endpoint reads no other input. It takes no save session and no character.

## Output

The endpoint returns a typed result:

```go
type AppearancePresetSummary struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Image    string   `json:"image"`
	BodyType string   `json:"bodyType"`
	Tags     []string `json:"tags"`
}

type GetAppearancePresetsResult struct {
	Presets []AppearancePresetSummary `json:"presets"`
}
```

| Field | Type | Meaning |
|---|---|---|
| `presets` | array of `AppearancePresetSummary` | The matching presets in catalog order. Always non-nil; empty when nothing matches. |
| `id` | `string` | Stable lowercase kebab-case identifier. It equals the `image` file name without the `.jpg` suffix. |
| `name` | `string` | Display name of the preset. |
| `image` | `string` | Asset file name inside `assets/appearance`, without a directory. |
| `bodyType` | `string` | `"Type A"` for the stored value `1`, `"Type B"` for the stored value `0`. |
| `tags` | array of `string` | Tags of the preset. Always non-nil; empty for every stored preset today. |

### What the getter deliberately does not return

The full appearance configuration stays inside `GameCatalog`. This getter never
returns:

- `voiceType`;
- the model identifiers (`faceModel`, `hairModel`, `eyeModel`, `eyebrowModel`,
  `beardModel`, `eyepatchModel`, `decalModel`, `eyelashModel`);
- `faceShape` (64 bytes), `body` (7 bytes), `skin` (91 bytes).

Those values are stored and validated by `GameCatalog` and are consumed by
`ApplyAppearancePreset`, which is a separate, mutating endpoint. A list consumer
receives metadata only.

## Stored presets

The file holds twenty presets. An unfiltered call returns all of them in exactly
the file order:

| # | `id` | `bodyType` |
|---|---|---|
| 1 | `geralt-of-rivia-the-witcher` | `Type A` |
| 2 | `sekiro-the-wolf-shinobi` | `Type A` |
| 3 | `ragnar-lodbrok-a-viking-warrior` | `Type A` |
| 4 | `trevor-belmont-vampire-hunter-from-castlevania` | `Type A` |
| 5 | `yennefer-sorceress-from-the-witcher` | `Type B` |
| 6 | `obi-wan-kenobi-a-jedi-master` | `Type A` |
| 7 | `lord-voldemort-the-dark-wizard` | `Type A` |
| 8 | `red-skull-a-mutated-humanoid` | `Type A` |
| 9 | `isaac-the-devil-forgemaster` | `Type A` |
| 10 | `thornkettle-the-forest-gnome` | `Type A` |
| 11 | `kratos-the-god-of-war` | `Type A` |
| 12 | `queen-marika-the-god-of-elden-ring` | `Type B` |
| 13 | `ciri-the-princess-of-cintra-from-witcher` | `Type B` |
| 14 | `makima-the-devil-hunter-from-chainsaw-man` | `Type B` |
| 15 | `melina-the-tarnished-finger-maiden` | `Type B` |
| 16 | `helga-the-tarnished-barbarian` | `Type B` |
| 17 | `witch-of-salem-the-blackflame-apostle` | `Type B` |
| 18 | `eleonora-the-sexy-twinblade-queen` | `Type B` |
| 19 | `casca-berserks-band-of-the-falcon-commander` | `Type B` |
| 20 | `fire-keeper-the-dark-souls-3-npc` | `Type B` |

Filtering never reorders the list: a filtered result is the same sequence with
non-matching entries removed.

None of the twenty presets declares a tag, because the migrated data declares
none. Any non-empty, syntactically valid tag filter therefore returns an empty
list today.

## Processing flow

1. A `nil` catalog is rejected before anything else is read.
2. Every supplied tag is checked; an empty tag ends the call with an error and
   an empty result.
3. The getter reads the presets from the loaded `GameCatalog`, which validated
   `presets/appearance.json` when it was built. A catalog built without
   appearance presets is an error, not an empty list.
4. Each preset is matched against `search` and then against `tags`.
5. A matching preset is reduced to its summary; the numeric body type is mapped
   to its public label.
6. The catalog returns an independent copy per call, so mutating one result never
   affects another call and never changes the catalog. The endpoint keeps no
   mutable shared state and re-reads no file.
7. The getter modifies nothing.

## Validation and errors

- An empty tag returns `tag <index> must not be empty` and an empty result. The
  HTTP route maps it to `400`, because the value comes from the client.
- A `nil` catalog returns `game catalog is not loaded`, and a catalog built
  without appearance presets returns `appearance presets are not loaded`. Both
  are backend configuration errors, not client input errors.
- There is no other validation. `GameCatalog` rejects an absent, malformed, or
  inconsistent `presets/appearance.json` while it is being built, including a
  duplicate identifier, name or image, an identifier that is not lowercase
  kebab-case, an `image` that is not `<id>.jpg` or carries a path, a `bodyType`
  outside `0`/`1`, a `voiceType` outside `0`–`5`, a value outside the `uint8`
  range, a `faceShape` that is not exactly 64 values, a `body` that is not
  exactly 7 values, a `skin` that is not exactly 91 values, an empty or duplicate
  tag, and a missing asset under `assets/appearance/<image>`.

## Dependencies

- The endpoint reads no save. It never opens a `.sl2`, a `.dat`, or a save
  session, and it performs no mutation of any kind.
- The endpoint reads exactly one thing: the appearance presets of the already
  loaded `GameCatalog`. It reads no manifest, no resource, and no other catalog
  data, it never opens `presets/appearance.json` itself, and it never loads or
  reloads the catalog.
- The endpoint calls no other endpoint and uses no shared endpoint helper.
- The endpoint does not import `backend/db`, `backend/db/data`, `backend/core`,
  or `internal/application`.

### Legacy source status

The legacy `backend/db/data/presets_generated.go` file and
`frontend/public/presets` assets were removed from the active tree during the
2.0 cutover. They remain available through Git history and the `v1.6.8` tag.
The values and images migrated into `presets/appearance.json` and
`assets/appearance` are the only data read by `GetAppearancePresets`.

## Command-line verification

`GetAppearancePresets` is exposed over HTTP as `GET /api/v1/appearance/presets`
by the local OpenAPI explorer in `backend/swagger`, a developer tool
the application neither imports nor starts. The route passes the catalog the
explorer loaded at start-up. A missing or invalid `presets/appearance.json`, or a
missing asset, stops the explorer while it loads its data, instead of leaving the
route half working.

The route reports the asset file name only. The explorer does not serve the
images yet; an asset route is a separate, later task.

### Run tests

From the repository root:

```bash
go test ./backend/endpoints/appearance -run '^TestGetAppearancePresets' -count=1 -v
```

### Call the route

Start the explorer:

```bash
go run ./backend/swagger
```

Then request every preset:

```bash
curl -s http://127.0.0.1:8788/api/v1/appearance/presets
```

```json
{
  "presets": [
    {
      "id": "geralt-of-rivia-the-witcher",
      "name": "Geralt of Rivia, the Witcher",
      "image": "geralt-of-rivia-the-witcher.jpg",
      "bodyType": "Type A",
      "tags": []
    }
  ]
}
```

The excerpt above is shortened: the real response carries all twenty summaries.

Filter by a substring of the identifier or the name:

```bash
curl -s "http://127.0.0.1:8788/api/v1/appearance/presets?search=witcher"
```

Filter by tags, supplied as one comma-separated value:

```bash
curl -s "http://127.0.0.1:8788/api/v1/appearance/presets?tags=witcher,female"
```

Because no stored preset declares a tag, that call returns an empty list:

```json
{
  "presets": []
}
```

An empty tag element is rejected with `400`:

```bash
curl -s "http://127.0.0.1:8788/api/v1/appearance/presets?tags=foo,,bar"
```

```json
{
  "error": "tag 1 must not be empty"
}
```

No demonstration program or helper script for this endpoint is kept in the
repository.

## Current limitations

- The endpoint is not exposed through Wails.
- The only HTTP route is `GET /api/v1/appearance/presets` of the local OpenAPI
  explorer in `backend/swagger`, a developer tool.
- There is no permanent CLI command for it.
- No route serves the preset images. The result carries the asset file name and
  a consumer cannot yet fetch the file over HTTP.
- The result carries list metadata only. The full appearance configuration is
  never returned by this getter.
- There is no paging, no sorting, no fuzzy search, and no favourites.
- The endpoint only reports presets. Applying one to a character
  (`ApplyAppearancePreset`) is a separate endpoint and is not implemented here.
