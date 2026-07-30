package schema

import "fmt"

func validateCapabilities(capabilities ItemCapabilities, sources map[SourceID]struct{}) error {
	if err := validateUpgradeCapability(capabilities.Upgrade, sources); err != nil {
		return err
	}
	if err := validateInfusionCapability(capabilities.Infusion, sources); err != nil {
		return err
	}
	if err := validateAshOfWarCapability(capabilities.AshOfWarMount, sources); err != nil {
		return err
	}
	if err := validateCapability("stack", capabilities.Stack, sources); err != nil {
		return err
	}
	if capabilities.Stack.Enabled && capabilities.Stack.Rules.MaxPerStack == 0 {
		return fmt.Errorf("stack: max per stack must be greater than zero")
	}
	if err := validateCapability("equipment", capabilities.Equipment, sources); err != nil {
		return err
	}
	if capabilities.Equipment.Enabled && len(capabilities.Equipment.Rules.AllowedSlots) == 0 {
		return fmt.Errorf("equipment: at least one slot is required")
	}
	return nil
}

func validateUpgradeCapability(capability Capability[UpgradeRules], sources map[SourceID]struct{}) error {
	if err := validateCapability("upgrade", capability, sources); err != nil {
		return err
	}
	if !capability.Enabled {
		return nil
	}
	if capability.Rules.Model != UpgradeModelStandard {
		return fmt.Errorf("upgrade: unsupported model %q", capability.Rules.Model)
	}
	if capability.Rules.MaxLevel == 0 {
		return fmt.Errorf("upgrade: max level must be greater than zero")
	}
	return nil
}

func validateInfusionCapability(capability Capability[InfusionRules], sources map[SourceID]struct{}) error {
	if err := validateCapability("infusion", capability, sources); err != nil {
		return err
	}
	if !capability.Enabled {
		return nil
	}
	if len(capability.Rules.AllowedAffinities) == 0 {
		return fmt.Errorf("infusion: at least one affinity is required")
	}
	seen := make(map[Affinity]struct{}, len(capability.Rules.AllowedAffinities))
	for _, affinity := range capability.Rules.AllowedAffinities {
		if !validAffinity(affinity) {
			return fmt.Errorf("infusion: unsupported affinity %q", affinity)
		}
		if _, exists := seen[affinity]; exists {
			return fmt.Errorf("infusion: duplicate affinity %q", affinity)
		}
		seen[affinity] = struct{}{}
	}
	return nil
}

func validateAshOfWarCapability(capability Capability[AshOfWarMountRules], sources map[SourceID]struct{}) error {
	if err := validateCapability("ashOfWarMount", capability, sources); err != nil {
		return err
	}
	if !capability.Enabled {
		return nil
	}
	if capability.Rules.Mode != AshOfWarMountModeCustom {
		return fmt.Errorf("ashOfWarMount: unsupported mode %q", capability.Rules.Mode)
	}
	if capability.Rules.WeaponType == "" {
		return fmt.Errorf("ashOfWarMount: weapon type is required")
	}
	return nil
}

func validateCapability[T any](name string, capability Capability[T], sources map[SourceID]struct{}) error {
	if err := validateProvenance(name, capability.Provenance, sources); err != nil {
		return err
	}
	if !capability.Known {
		if capability.Enabled || capability.Rules != nil {
			return fmt.Errorf("%s: unknown capability cannot be enabled or contain rules", name)
		}
		return nil
	}
	if capability.Enabled && capability.Rules == nil {
		return fmt.Errorf("%s: enabled capability requires rules", name)
	}
	if !capability.Enabled && capability.Rules != nil {
		return fmt.Errorf("%s: disabled capability cannot contain rules", name)
	}
	return nil
}

func validAffinity(affinity Affinity) bool {
	switch affinity {
	case AffinityStandard,
		AffinityHeavy,
		AffinityKeen,
		AffinityQuality,
		AffinityFire,
		AffinityFlameArt,
		AffinityLightning,
		AffinitySacred,
		AffinityMagic,
		AffinityCold,
		AffinityPoison,
		AffinityBlood,
		AffinityOccult:
		return true
	default:
		return false
	}
}
