package inventory

import (
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

// commonItemAddition is the catalog-owned part of the common-container add
// contract shared by the capacity getter and common item-add endpoints. SaveEngine
// still owns every physical read and write.
type commonItemAddition struct {
	resource          schema.Resource
	gameID            uint32
	separateInstances bool
	maxPerStack       uint32
}

// resolveCommonItemAddition proves the facts that do not depend on the target
// container. The two supported families have ID-derived handles and therefore
// need no variable-length GaItem allocation. Ambiguous key routing is rejected
// once here for every common-only consumer.
func resolveCommonItemAddition(
	gameCatalog *gamecatalog.Catalog,
	kind string,
	key string,
	variantID *uint32,
) (commonItemAddition, error) {
	resource, err := gameCatalog.ResourceByKindKeyAndVariant(schema.ResourceKind(kind), key, variantID)
	if err != nil {
		return commonItemAddition{}, err
	}
	if resource.Item == nil {
		return commonItemAddition{}, fmt.Errorf(
			"resource kind %q key %q has no item document", kind, key)
	}
	item := resource.Item

	if !item.Family.Known {
		return commonItemAddition{}, fmt.Errorf(
			"resource kind %q key %q has an unknown family", kind, key)
	}
	var familyPrefix uint32
	switch item.Family.Value {
	case schema.ItemFamilyGoods:
		familyPrefix = addItemGoodsPrefix
	case schema.ItemFamilyTalisman:
		familyPrefix = addItemTalismanPrefix
	default:
		return commonItemAddition{}, fmt.Errorf(
			"resource kind %q key %q is of family %q; common item addition supports only %q and %q",
			kind, key, item.Family.Value, schema.ItemFamilyGoods, schema.ItemFamilyTalisman)
	}

	if !item.GameID.Known {
		return commonItemAddition{}, fmt.Errorf(
			"resource kind %q key %q has an unknown game ID", kind, key)
	}
	gameID := item.GameID.Value
	if gameID&addItemFamilyPrefix != familyPrefix {
		return commonItemAddition{}, fmt.Errorf(
			"resource kind %q key %q declares family %q and game ID 0x%08X, which disagree",
			kind, key, item.Family.Value, gameID)
	}

	if !item.Category.Known {
		return commonItemAddition{}, fmt.Errorf(
			"resource kind %q key %q has an unknown category", kind, key)
	}
	if item.Category.Value == addItemKeyCategory {
		return commonItemAddition{}, fmt.Errorf(
			"resource kind %q key %q is in category %q, which does not distinguish common from"+
				" key routing; common-only operations reject the category fail-closed",
			kind, key, addItemKeyCategory)
	}

	if !item.Storage.RecordMode.Known {
		return commonItemAddition{}, fmt.Errorf(
			"resource kind %q key %q has an unknown record mode", kind, key)
	}
	resolved := commonItemAddition{resource: resource, gameID: gameID}
	switch item.Storage.RecordMode.Value {
	case schema.RecordModeQuantityStack:
		stack := item.Capabilities.Stack
		if !stack.Known {
			return commonItemAddition{}, fmt.Errorf(
				"resource kind %q key %q has an unknown stack capability", kind, key)
		}
		if !stack.Enabled {
			return commonItemAddition{}, fmt.Errorf(
				"resource kind %q key %q stores a quantity but does not stack", kind, key)
		}
		if stack.Rules == nil || stack.Rules.MaxPerStack == 0 {
			return commonItemAddition{}, fmt.Errorf(
				"resource kind %q key %q carries no stack limit", kind, key)
		}
		resolved.maxPerStack = stack.Rules.MaxPerStack
	case schema.RecordModeSeparateInstances:
		resolved.separateInstances = true
		resolved.maxPerStack = 1
	default:
		return commonItemAddition{}, fmt.Errorf(
			"resource kind %q key %q declares the unsupported record mode %q",
			kind, key, item.Storage.RecordMode.Value)
	}
	return resolved, nil
}
