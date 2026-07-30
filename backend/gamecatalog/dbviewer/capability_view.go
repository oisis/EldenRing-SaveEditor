package dbviewer

import (
	"fmt"
	"strings"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

type capabilityView struct {
	Name           string
	State          string
	StateClass     string
	Details        []string
	Source         schema.SourceID
	Method         string
	SourceLocation string
}

func (server *Server) capabilityViews(item *schema.ItemDocument) []capabilityView {
	return []capabilityView{
		server.capability("Upgrade", item.Capabilities.Upgrade.Known, item.Capabilities.Upgrade.Enabled,
			upgradeDetails(item.Capabilities.Upgrade.Rules), item.Capabilities.Upgrade.Provenance),
		server.capability("Infusion", item.Capabilities.Infusion.Known, item.Capabilities.Infusion.Enabled,
			infusionDetails(item.Capabilities.Infusion.Rules), item.Capabilities.Infusion.Provenance),
		server.capability("Ash of War mount", item.Capabilities.AshOfWarMount.Known, item.Capabilities.AshOfWarMount.Enabled,
			ashOfWarMountDetails(item.Capabilities.AshOfWarMount.Rules), item.Capabilities.AshOfWarMount.Provenance),
		server.capability("Stack", item.Capabilities.Stack.Known, item.Capabilities.Stack.Enabled,
			stackDetails(item.Capabilities.Stack.Rules), item.Capabilities.Stack.Provenance),
		server.capability("Equipment", item.Capabilities.Equipment.Known, item.Capabilities.Equipment.Enabled,
			equipmentDetails(item.Capabilities.Equipment.Rules), item.Capabilities.Equipment.Provenance),
	}
}

func (server *Server) capability(
	name string,
	known bool,
	enabled bool,
	details []string,
	provenance schema.Provenance,
) capabilityView {
	state, class := "Disabled", "disabled"
	if !known {
		state, class = "Unknown", "unknown"
		details = []string{"Action must fail closed."}
	} else if enabled {
		state, class = "Enabled", "enabled"
	}
	source := server.sources[provenance.Source]
	return capabilityView{
		Name:           name,
		State:          state,
		StateClass:     class,
		Details:        details,
		Source:         provenance.Source,
		Method:         provenance.Method,
		SourceLocation: source.Location,
	}
}

func upgradeDetails(rules *schema.UpgradeRules) []string {
	if rules == nil {
		return nil
	}
	return []string{fmt.Sprintf("Model: %s", rules.Model), fmt.Sprintf("Maximum level: +%d", rules.MaxLevel)}
}

func infusionDetails(rules *schema.InfusionRules) []string {
	if rules == nil {
		return nil
	}
	values := make([]string, 0, len(rules.AllowedAffinities))
	for _, affinity := range rules.AllowedAffinities {
		values = append(values, string(affinity))
	}
	return []string{"Allowed: " + strings.Join(values, ", ")}
}

func ashOfWarMountDetails(rules *schema.AshOfWarMountRules) []string {
	if rules == nil {
		return nil
	}
	return []string{
		fmt.Sprintf("Mode: %s", rules.Mode),
		fmt.Sprintf("Weapon type: %s", rules.WeaponType),
		fmt.Sprintf("Compatibility bit: %d", rules.CompatibilityBit),
	}
}

func stackDetails(rules *schema.StackRules) []string {
	if rules == nil {
		return nil
	}
	return []string{fmt.Sprintf("Maximum per stack: %d", rules.MaxPerStack)}
}

func equipmentDetails(rules *schema.EquipmentRules) []string {
	if rules == nil {
		return nil
	}
	slots := make([]string, 0, len(rules.AllowedSlots))
	for _, slot := range rules.AllowedSlots {
		slots = append(slots, string(slot))
	}
	return []string{"Allowed slots: " + strings.Join(slots, ", ")}
}
