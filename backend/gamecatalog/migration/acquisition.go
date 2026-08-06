package migration

import "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"

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
		acquisition.RequiredContainerID = notApplicableCatalogFact[uint32](
			"spirit ash items are never held in a legacy RequiredContainer",
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
