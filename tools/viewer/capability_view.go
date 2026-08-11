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
	return server.capabilityViewsFor(item.Capabilities)
}

func (server *Server) capabilityViewsFor(capabilities schema.ItemCapabilities) []capabilityView {
	return []capabilityView{
		server.capability("Upgrade", capabilities.Upgrade.Known, capabilities.Upgrade.Enabled,
			upgradeDetails(capabilities.Upgrade.Rules), capabilities.Upgrade.Provenance),
		server.capability("Infusion", capabilities.Infusion.Known, capabilities.Infusion.Enabled,
			infusionDetails(capabilities.Infusion.Rules), capabilities.Infusion.Provenance),
		server.capability("Ash of War mount", capabilities.AshOfWarMount.Known, capabilities.AshOfWarMount.Enabled,
			ashOfWarMountDetails(capabilities.AshOfWarMount.Rules), capabilities.AshOfWarMount.Provenance),
		server.capability("Stack", capabilities.Stack.Known, capabilities.Stack.Enabled,
			stackDetails(capabilities.Stack.Rules), capabilities.Stack.Provenance),
		server.capability("Equipment", capabilities.Equipment.Known, capabilities.Equipment.Enabled,
			equipmentDetails(capabilities.Equipment.Rules), capabilities.Equipment.Provenance),
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
	details := []string{
		fmt.Sprintf("Model: %s", rules.Model),
		fmt.Sprintf("Maximum level: +%d", rules.MaxLevel),
	}
	if rules.MaxLevelSFV != nil {
		details = append(
			details,
			fmt.Sprintf(
				"Maximum level — SaveForge value: +%d",
				rules.MaxLevelSFV.Value,
			),
		)
	}
	return details
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
