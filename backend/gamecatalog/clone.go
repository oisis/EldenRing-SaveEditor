package gamecatalog

import "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"

func cloneManifest(manifest schema.Manifest) schema.Manifest {
	cloned := manifest
	cloned.Sources = append([]schema.DataSource(nil), manifest.Sources...)
	return cloned
}

func cloneResource(resource schema.Resource) schema.Resource {
	cloned := resource
	if resource.Item == nil {
		return cloned
	}

	item := *resource.Item
	item.Variants = append([]schema.ItemVariant(nil), resource.Item.Variants...)
	item.Capabilities.Upgrade = cloneCapability(resource.Item.Capabilities.Upgrade, nil)
	item.Capabilities.Infusion = cloneCapability(
		resource.Item.Capabilities.Infusion,
		func(rules schema.InfusionRules) schema.InfusionRules {
			rules.AllowedAffinities = append([]schema.Affinity(nil), rules.AllowedAffinities...)
			return rules
		},
	)
	item.Capabilities.AshOfWarMount = cloneCapability(resource.Item.Capabilities.AshOfWarMount, nil)
	item.Capabilities.Stack = cloneCapability(resource.Item.Capabilities.Stack, nil)
	item.Capabilities.Equipment = cloneCapability(
		resource.Item.Capabilities.Equipment,
		func(rules schema.EquipmentRules) schema.EquipmentRules {
			rules.AllowedSlots = append([]schema.EquipmentSlot(nil), rules.AllowedSlots...)
			return rules
		},
	)
	if resource.Item.Weapon != nil {
		weapon := *resource.Item.Weapon
		item.Weapon = &weapon
	}
	if resource.Item.AshOfWar != nil {
		ash := *resource.Item.AshOfWar
		item.AshOfWar = &ash
	}
	cloned.Item = &item
	return cloned
}

func cloneCapability[T any](capability schema.Capability[T], cloneRules func(T) T) schema.Capability[T] {
	cloned := capability
	if capability.Rules == nil {
		return cloned
	}
	rules := *capability.Rules
	if cloneRules != nil {
		rules = cloneRules(rules)
	}
	cloned.Rules = &rules
	return cloned
}

func cloneRelations(relations []schema.Relation) []schema.Relation {
	return append([]schema.Relation(nil), relations...)
}
