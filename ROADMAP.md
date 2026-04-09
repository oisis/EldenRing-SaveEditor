# ROADMAP

## Phase 22 — Item Descriptions & Stats

### Goal
Display item flavor text (in-game descriptions) and detailed stats (damage, weight, scaling, requirements, resistances) in the item detail modal alongside the enlarged icon.

### Data Sources (researched)

| Source | URL | Data | Format | Quality |
|---|---|---|---|---|
| **ERDB** (recommended) | https://github.com/EldenRingDatabase/erdb | Descriptions, full stats, effects | Pre-built JSON | Authoritative (parsed from game params) |
| Carian Archive | https://github.com/AsteriskAmpersand/Carian-Archive | All in-game text | JSON/XML | Complete text dump, no stats |
| elden-ring-data/msg | https://github.com/elden-ring-data/msg | Title-description pairs | JSON | Text only, well-structured |
| Elden Ring Fan API | https://eldenring.fanapis.com/ ([docs](https://docs.eldenring.fanapis.com/docs/)) | Items, descriptions, images | REST JSON (no auth) | Wiki-scraped, may lag behind DLC |
| regulation.bin extraction | Game files (Yapped Rune Bear) | Full game params + FMG text | Binary → JSON | Requires game + Windows tools |

### Recommendation
Use **ERDB** as primary source — MIT-licensed, contains both descriptions and stats per item, parsed directly from game files (regulation.bin). Supplement with **Carian Archive** if any text coverage gaps.

### Implementation Plan

1. **Data import** — Download ERDB JSON data, write a script (`scripts/import_descriptions.go`) to merge descriptions and stats into our existing `backend/db/data/*.go` maps. Add new fields to `ItemData` struct:
   - `Description string` — in-game flavor text
   - `Stats map[string]interface{}` — damage, weight, scaling, requirements (structure varies by category)

2. **Backend** — Extend `db.GetItemData()` and `ItemEntry` to include description and stats. No new endpoints needed — existing item data flow already passes through `ItemViewModel`.

3. **Frontend** — Extend the item detail modal (enlarged icon view) to show:
   - Description text below the icon
   - Stats table (collapsible, category-aware: weapons show damage/scaling, armor shows resistances, consumables show effects)

4. **Build-time vs runtime** — Embed descriptions in Go data files at build time (like current item names). No runtime API calls — keeps the app fully offline.

### Open Questions
- ERDB item IDs may not match our DB IDs 1:1 — need a mapping step during import
- Stats structure differs per category (weapon vs armor vs consumable) — need flexible schema
- Text localization: ERDB supports multiple languages — start with English only?
