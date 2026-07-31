package migration

import (
	"sort"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func (context *generationContext) buildSafety(item seed) schema.ItemSafety {
	flags := item.Flags
	if item.Category == "gestures" {
		flags = mergeGestureFlags(flags, context.gesturesByItem[item.ID])
	}
	method := "derived from legacy ItemData flags"
	if item.RegulationOnlyVariant {
		method = "derived from the canonical legacy ItemData flags for a Regulation-only variant"
	} else if !item.HasLegacyItem {
		method = "derived exclusively from legacy AllGestures flags"
	}
	return safetyFromFlagsWithMethod(flags, method)
}

func safetyFromFlags(flags []string) schema.ItemSafety {
	return safetyFromFlagsWithMethod(flags, "derived from legacy item flags")
}

func safetyFromFlagsWithMethod(flags []string, method string) schema.ItemSafety {
	return schema.ItemSafety{
		CutContent:   knownLegacyFact(hasLegacyFlag(flags, "cut_content"), method),
		BanRisk:      knownLegacyFact(hasLegacyFlag(flags, "ban_risk"), method),
		DLC:          knownLegacyFact(hasLegacyFlag(flags, "dlc"), method),
		NoDatabase:   knownLegacyFact(hasLegacyFlag(flags, "no_database"), method),
		ScalesWithNG: knownLegacyFact(hasLegacyFlag(flags, "scales_with_ng"), method),
	}
}

func hasLegacyFlag(flags []string, target string) bool {
	for _, flag := range flags {
		if flag == target {
			return true
		}
	}
	return false
}

func mergeGestureFlags(base []string, slots []gestureSlotSeed) []string {
	seen := make(map[string]struct{})
	for _, flag := range base {
		seen[flag] = struct{}{}
	}
	for _, slot := range slots {
		for _, flag := range slot.Flags {
			seen[flag] = struct{}{}
		}
	}
	result := make([]string, 0, len(seen))
	for flag := range seen {
		result = append(result, flag)
	}
	sort.Strings(result)
	return result
}
