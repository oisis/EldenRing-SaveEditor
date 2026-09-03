package safetyprofile

import (
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

// This file is the one implementation of the profile rules. Nothing here reads
// a save, a session or a setting file: it answers questions about one catalog
// item under one profile, and every caller supplies both.
//
// The two "-sfv" fields of ItemStorage are deliberately never read. They are
// SaveForge-verified research values, not runtime limits, and turning one into
// a limit here would silently change what the game accepts.

// InventoryLimit reports the container limit of one item under profile.
//
// Under Safe the catalog's safeModeMaxInventory is used when the item declares
// that field; when it does not, the base maxInventory applies, because a
// missing Safe Mode field is an absent narrower rule and never a zero limit.
// Both other profiles always use the base limit.
//
// known false means the catalog states no usable limit at all. The caller
// rejects the operation rather than defaulting one.
func InventoryLimit(profile Profile, item *schema.ItemDocument) (limit uint32, known bool) {
	if item == nil {
		return 0, false
	}
	if profile == Safe {
		if safe, ok := knownFact(item.Storage.SafeModeMaxInventory); ok {
			return safe, true
		}
	}
	if !item.Storage.MaxInventory.Known {
		return 0, false
	}
	return item.Storage.MaxInventory.Value, true
}

// StorageLimit is InventoryLimit for the Storage Box, under the same rules.
func StorageLimit(profile Profile, item *schema.ItemDocument) (limit uint32, known bool) {
	if item == nil {
		return 0, false
	}
	if profile == Safe {
		if safe, ok := knownFact(item.Storage.SafeModeMaxStorage); ok {
			return safe, true
		}
	}
	if !item.Storage.MaxStorage.Known {
		return 0, false
	}
	return item.Storage.MaxStorage.Value, true
}

// HiddenFromItemDatabase reports whether the general Item Database must not
// present one item under profile.
//
// noDatabase is unconditional: such an item is reserved for the feature that
// owns it and stays hidden under every profile, Chaos included. banRisk and
// cutContent are hidden under Safe and Expanded Limits and revealed under
// Chaos. Only a known true hides a resource, so undecided catalog data never
// silently removes an item from the list.
//
// dlc and preOrder are presentation facts and never hide anything.
func HiddenFromItemDatabase(profile Profile, item *schema.ItemDocument) bool {
	if item == nil {
		return false
	}
	return HiddenFromItemDatabaseFlags(
		profile,
		knownTrue(item.Safety.NoDatabase),
		knownTrue(item.Safety.BanRisk),
		knownTrue(item.Safety.CutContent),
	)
}

// RequiresBanRiskConfirmation reports whether adding one item has to be
// confirmed explicitly by the user. It is a property of the item alone: the
// confirmation is required under every profile that can reach the item at all.
func RequiresBanRiskConfirmation(item *schema.ItemDocument) bool {
	return item != nil && knownTrue(item.Safety.BanRisk)
}

// AllowMutation rejects an operation on one item that the active profile does
// not permit. It is the backend gate a call that bypasses the interface still
// has to pass, so it repeats no interface decision and trusts no caller flag.
//
// confirmedBanRisk is the user's explicit confirmation for this exact
// operation. It is required whenever the item carries a known ban risk, and it
// can never substitute for a profile that hides the item in the first place.
func AllowMutation(profile Profile, item *schema.ItemDocument, confirmedBanRisk bool) error {
	if item == nil {
		return fmt.Errorf("the resource carries no item document")
	}
	if profile != Chaos {
		if knownTrue(item.Safety.BanRisk) {
			return fmt.Errorf(
				"the resource is marked ban risk and the active safety profile %q does not allow it",
				profile)
		}
		if knownTrue(item.Safety.CutContent) {
			return fmt.Errorf(
				"the resource is cut content and the active safety profile %q does not allow it",
				profile)
		}
	}
	if RequiresBanRiskConfirmation(item) && !confirmedBanRisk {
		return fmt.Errorf("the resource is marked ban risk and needs an explicit confirmation")
	}
	return nil
}

func knownFact(fact *schema.Fact[uint32]) (uint32, bool) {
	if fact == nil || !fact.Known {
		return 0, false
	}
	return fact.Value, true
}

func knownTrue(fact schema.Fact[bool]) bool {
	return fact.Known && fact.Value
}

// HiddenFromItemDatabaseFlags is HiddenFromItemDatabase for a caller that holds
// the three safety flags without the document they were read from, such as a
// list built from scalar catalog summaries. Both entry points share this one
// implementation, so the visibility rule exists exactly once.
func HiddenFromItemDatabaseFlags(profile Profile, noDatabase, banRisk, cutContent bool) bool {
	if noDatabase {
		return true
	}
	if profile == Chaos {
		return false
	}
	return banRisk || cutContent
}
