package migration

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestGenerateBuildsEverySpiritAshUpgradeFromRegulationChain(t *testing.T) {
	catalog, err := Generate(localGenerateOptions(t))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	canonical := 0
	variants := 0
	grave := 0
	ghost := 0
	for _, resource := range catalog.Resources {
		item := resource.Item
		if item == nil || item.Family.Value != schema.ItemFamilySpiritAsh {
			continue
		}
		canonical++
		variants += len(item.Variants)
		upgrade := item.Capabilities.Upgrade
		if !upgrade.Known || !upgrade.Enabled || upgrade.Rules == nil {
			t.Fatalf("Spirit Ash 0x%08X has disabled upgrade capability", item.GameID.Value)
		}
		if upgrade.Rules.MaxLevel != 10 {
			t.Fatalf(
				"Spirit Ash 0x%08X max upgrade = %d, want 10",
				item.GameID.Value,
				upgrade.Rules.MaxLevel,
			)
		}
		switch upgrade.Rules.Model {
		case schema.UpgradeModelGraveGlovewort:
			grave++
		case schema.UpgradeModelGhostGlovewort:
			ghost++
		default:
			t.Fatalf(
				"Spirit Ash 0x%08X model = %q",
				item.GameID.Value,
				upgrade.Rules.Model,
			)
		}
	}
	if canonical != 84 || variants != 840 || grave != 51 || ghost != 33 {
		t.Fatalf(
			"Spirit Ash canonical/variants/grave/ghost = %d/%d/%d/%d, want 84/840/51/33",
			canonical,
			variants,
			grave,
			ghost,
		)
	}
}
