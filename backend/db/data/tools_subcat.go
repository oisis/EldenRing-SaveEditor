package data

// tools_subcat.go — sub-category assignment for the Tools tab.
//
// Sub-groups (in-game order, 12 total):
//   1.  Sacred Flasks, Reusable Tools & FP Regenerators
//   2.  Consumables
//   3.  Throwing Pots
//   4.  Perfume Arts
//   5.  Throwables
//   6.  Catalyst Tools  (overlay-buff one-shots — name-based lookup)
//   7.  Grease
//   8.  Miscellaneous Tools
//   9.  Quest Tools
//   10. Golden Runes  (already assigned inline in tools.go)
//   11. Remembrances
//   12. Multiplayer Items
//
// Classification uses the IconPath sub-folder as the primary signal. A small
// name-based override list catches Catalyst Tools (no dedicated icon dir).
// The Golden Runes block was already assigned `SubcatToolsGoldenRunes` when
// it was relocated from bolstering_materials.go (commit afca994); init()
// skips entries that already have SubCategory set.
//
// PHASE-2 NOTE: when icons are flattened from `items/tools/<sub>/X.png` →
// `items/tools/X.png`, the IconPath-based classification will stop working.
// At that point, inline SubCategory into each entry of tools.go and remove
// this file, OR replace the IconPath-based switch with a name-based one.

import "strings"

// catalystToolsByName — items that belong in "Catalyst Tools" sub-group,
// identified by exact name (no dedicated icon sub-folder).
var catalystToolsByName = map[string]struct{}{
	// Spell-amplifying overlay tools — additive buff, one-shot use.
	// Tagged conservatively; expand as new candidates are identified.
	"Stimulating Boluses": {},
}

// iconPathSubdirToToolsSubcat maps the second path component (after `tools/`)
// to a sub-category constant.
var iconPathSubdirToToolsSubcat = map[string]string{
	"sacred_flasks": SubcatToolsFlasks,
	"consumables":   SubcatToolsConsumables,
	"pots":          SubcatToolsThrowingPots,
	"perfume":       SubcatToolsPerfumeArts,
	"throwables":    SubcatToolsThrowables,
	"grease":        SubcatToolsGrease,
	"misc":          SubcatToolsMisc,
	"quest":         SubcatToolsQuest,
	"runes":         SubcatToolsGoldenRunes,
	"remembrances":  SubcatToolsRemembrances,
	"multiplayer":   SubcatToolsMultiplayer,
}

// classifyTool returns the SubCategory for a Tools entry based on IconPath
// sub-folder, with a name-based override for Catalyst Tools.
func classifyTool(item ItemData) string {
	if _, ok := catalystToolsByName[item.Name]; ok {
		return SubcatToolsCatalystTools
	}
	const prefix = "items/tools/"
	if !strings.HasPrefix(item.IconPath, prefix) {
		return SubcatToolsMisc
	}
	rest := item.IconPath[len(prefix):]
	slash := strings.Index(rest, "/")
	if slash <= 0 {
		// Flat path (no sub-folder) — fall through to misc.
		return SubcatToolsMisc
	}
	subdir := rest[:slash]
	if sc, ok := iconPathSubdirToToolsSubcat[subdir]; ok {
		return sc
	}
	return SubcatToolsMisc
}

func init() {
	for id, item := range Tools {
		if item.SubCategory != "" {
			continue
		}
		item.SubCategory = classifyTool(item)
		Tools[id] = item
	}
}
