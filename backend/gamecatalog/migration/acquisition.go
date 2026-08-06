package migration

import "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"

const fireKnightGreatswordName = "Fire Knight's Greatsword"

func requiredContainerIsNotApplicable(item seed, family schema.ItemFamily) bool {
	return family == schema.ItemFamilySpiritAsh ||
		(family == schema.ItemFamilyWeapon && item.Name == fireKnightGreatswordName)
}

func buildAcquisition(item seed, requiredContainerNotApplicable bool) schema.ItemAcquisition {
	acquisition := schema.ItemAcquisition{
		IsContainer: knownLegacyFact(
			item.Acquisition.IsContainer,
			"copied from legacy ContainerItemIDs",
		),
		ContainerPickupFlagIDs: knownLegacyFact(
			cloneUint32s(item.Acquisition.ContainerPickupFlags),
			"copied from legacy ContainerPickupFlags",
		),
		ContainerVendorFlagIDs: knownLegacyFact(
			cloneUint32s(item.Acquisition.ContainerVendorFlags),
			"copied from legacy ContainerVendorPurchaseFlags",
		),
		BolsteringPickupFlagIDs: knownLegacyFact(
			cloneUint32s(item.Acquisition.BolsteringPickupFlags),
			"copied from legacy BolsteringPickupFlags",
		),
		CompanionEventFlagIDs: knownLegacyFact(
			cloneUint32s(item.Acquisition.CompanionEventFlagIDs),
			"copied from legacy item companion event flags",
		),
	}
	if item.Acquisition.RequiredContainerID != nil {
		acquisition.RequiredContainerID = knownLegacyFact(
			*item.Acquisition.RequiredContainerID,
			"copied from legacy RequiredContainer",
		)
	} else if requiredContainerNotApplicable {
		reason := "spirit ash items are never held in a legacy RequiredContainer"
		if item.Name == fireKnightGreatswordName {
			reason = "Fire Knight's Greatsword is never held in a legacy RequiredContainer"
		}
		acquisition.RequiredContainerID = notApplicableCatalogFact[uint32](
			reason,
		)
	} else {
		acquisition.RequiredContainerID = unknownCatalogFact[uint32](
			"legacy RequiredContainer has no entry for this item",
		)
	}
	if item.Acquisition.WorldPickupFlagID != nil {
		acquisition.WorldPickupFlagID = knownLegacyFact(
			*item.Acquisition.WorldPickupFlagID,
			"copied from legacy WorldPickupFlagID",
		)
	} else {
		acquisition.WorldPickupFlagID = unknownCatalogFact[uint32](
			"legacy WorldPickupFlagID has no entry for this item",
		)
	}
	return acquisition
}
