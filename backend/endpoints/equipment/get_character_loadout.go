/*
Endpoint: GetCharacterLoadout
EndpointID: get_character_loadout
Purpose: Returns one coherent, catalog-resolved read-only loadout of a character slot.
How it works: SaveEngine reads all confirmed loadout groups under one session lock and validates cross-structure references. The endpoint then resolves every occupied game ID through GameCatalog and projects only domain presentation required by loadout consumers.
Supported resource types: ItemDocument of the family required by each loadout group.
Input variables: saveSessionID, characterID.
GameCatalog variables read: item family, presentation name, presentation iconPath, spell memorySlots and canonical ResourceRef.
Save variables read: equipment, quick items, pouch, Physick, spells, active indexes, Inventory common references, the effective Memory Stone count, unlocked talisman slots and current saveRevision.
Implementation status: implemented
*/
package equipment

import (
	"errors"
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/contract"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

const GetCharacterLoadoutEndpointID = "get_character_loadout"

var GetCharacterLoadoutDefinition = contract.MustDefine(contract.Definition{
	Name:                       "GetCharacterLoadout",
	ID:                         GetCharacterLoadoutEndpointID,
	Kind:                       contract.Getter,
	SupportedResourceTypes:     "ItemDocument of the family required by each loadout group",
	SupportedResourceVariables: []string{"saveSessionID", "characterID"},
	Description:                "Returns one coherent, catalog-resolved read-only loadout of a character slot.",
})

type LoadoutSlotState string

const (
	LoadoutSlotEmpty    LoadoutSlotState = "empty"
	LoadoutSlotOccupied LoadoutSlotState = "occupied"
	LoadoutSlotLocked   LoadoutSlotState = "locked"
)

type LoadoutSlot struct {
	SlotType schema.EquipmentSlot `json:"slotType"`
	State    LoadoutSlotState     `json:"state"`
	// OwnedItemID is the exact, revision-scoped Inventory common identity the
	// occupied position references, so the hand, armor and talisman setters can
	// be called with the record that is actually equipped. It is present only
	// for those three groups when the position is occupied: an empty position,
	// a locked position, an ammunition position and a Physick position carry
	// none. It is never derived from a game ID and never minted by a caller.
	OwnedItemID string              `json:"ownedItemID,omitempty"`
	Resource    *schema.ResourceRef `json:"resource,omitempty"`
	Name        string              `json:"name,omitempty"`
	IconPath    string              `json:"iconPath,omitempty"`
	RawValue    uint32              `json:"rawValue"`
}

type LoadoutOwnedSlot struct {
	SlotType    schema.EquipmentSlot `json:"slotType"`
	State       LoadoutSlotState     `json:"state"`
	OwnedItemID string               `json:"ownedItemID,omitempty"`
	Resource    *schema.ResourceRef  `json:"resource,omitempty"`
	Name        string               `json:"name,omitempty"`
	IconPath    string               `json:"iconPath,omitempty"`
	Quantity    uint32               `json:"quantity,omitempty"`
}

type LoadoutSpellSlot struct {
	State       LoadoutSlotState    `json:"state"`
	Resource    *schema.ResourceRef `json:"resource,omitempty"`
	Name        string              `json:"name,omitempty"`
	IconPath    string              `json:"iconPath,omitempty"`
	MemorySlots int                 `json:"memorySlots,omitempty"`
}

type GetCharacterLoadoutResult struct {
	SaveSessionID string `json:"saveSessionID"`
	SaveRevision  string `json:"saveRevision"`
	CharacterID   int    `json:"characterID"`
	Active        bool   `json:"active"`

	RightHand []LoadoutSlot `json:"rightHand"`
	LeftHand  []LoadoutSlot `json:"leftHand"`
	Arrows    []LoadoutSlot `json:"arrows"`
	Bolts     []LoadoutSlot `json:"bolts"`
	Armor     []LoadoutSlot `json:"armor"`
	Talismans []LoadoutSlot `json:"talismans"`

	QuickItems      []LoadoutOwnedSlot `json:"quickItems"`
	Pouch           []LoadoutOwnedSlot `json:"pouch"`
	ActiveQuickItem int32              `json:"activeQuickItem"`

	Physick []LoadoutSlot `json:"physick"`

	Spells                []LoadoutSpellSlot `json:"spells"`
	ActiveSpellIndex      int                `json:"activeSpellIndex"`
	UsedMemorySlots       int                `json:"usedMemorySlots"`
	AvailableMemorySlots  int                `json:"availableMemorySlots"`
	MemoryStones          uint32             `json:"memoryStones"`
	UnlockedTalismanSlots int                `json:"unlockedTalismanSlots"`
}

// GetCharacterLoadout returns one domain loadout snapshot. Unknown catalog
// identities and structurally invalid slot values fail closed; technical empty
// records are recognised before catalog resolution and never appear occupied.
func GetCharacterLoadout(
	engine *saveengine.Engine,
	gameCatalog *gamecatalog.Catalog,
	saveSessionID string,
	characterID int,
) (GetCharacterLoadoutResult, error) {
	if engine == nil {
		return GetCharacterLoadoutResult{}, errors.New("save engine is not available")
	}
	if gameCatalog == nil {
		return GetCharacterLoadoutResult{}, errors.New("game catalog is not available")
	}

	stored, err := engine.GetCharacterLoadoutSnapshot(saveSessionID, characterID)
	if err != nil {
		return GetCharacterLoadoutResult{}, err
	}
	result := emptyCharacterLoadout(stored)
	if !stored.Active {
		return result, nil
	}

	result.RightHand, err = resolveLoadoutGroup(gameCatalog, stored, []loadoutPosition{
		{index: 1, slot: schema.EquipmentSlotRightHand},
		{index: 3, slot: schema.EquipmentSlotRightHand},
		{index: 5, slot: schema.EquipmentSlotRightHand},
	}, schema.ItemFamilyWeapon)
	if err != nil {
		return GetCharacterLoadoutResult{}, fmt.Errorf("right hand: %w", err)
	}
	result.LeftHand, err = resolveLoadoutGroup(gameCatalog, stored, []loadoutPosition{
		{index: 0, slot: schema.EquipmentSlotLeftHand},
		{index: 2, slot: schema.EquipmentSlotLeftHand},
		{index: 4, slot: schema.EquipmentSlotLeftHand},
	}, schema.ItemFamilyWeapon)
	if err != nil {
		return GetCharacterLoadoutResult{}, fmt.Errorf("left hand: %w", err)
	}
	result.Arrows, err = resolveLoadoutGroup(gameCatalog, stored, []loadoutPosition{
		{index: 6, slot: schema.EquipmentSlotArrow},
		{index: 8, slot: schema.EquipmentSlotArrow},
	}, schema.ItemFamilyWeapon)
	if err != nil {
		return GetCharacterLoadoutResult{}, fmt.Errorf("arrows: %w", err)
	}
	result.Bolts, err = resolveLoadoutGroup(gameCatalog, stored, []loadoutPosition{
		{index: 7, slot: schema.EquipmentSlotBolt},
		{index: 9, slot: schema.EquipmentSlotBolt},
	}, schema.ItemFamilyWeapon)
	if err != nil {
		return GetCharacterLoadoutResult{}, fmt.Errorf("bolts: %w", err)
	}
	result.Armor, err = resolveLoadoutGroup(gameCatalog, stored, []loadoutPosition{
		{index: 12, slot: schema.EquipmentSlotHead},
		{index: 13, slot: schema.EquipmentSlotChest},
		{index: 14, slot: schema.EquipmentSlotArms},
		{index: 15, slot: schema.EquipmentSlotLegs},
	}, schema.ItemFamilyArmor)
	if err != nil {
		return GetCharacterLoadoutResult{}, fmt.Errorf("armor: %w", err)
	}
	result.Talismans, err = resolveTalismanGroup(gameCatalog, stored)
	if err != nil {
		return GetCharacterLoadoutResult{}, err
	}
	result.QuickItems, err = resolveOwnedLoadoutGroup(
		gameCatalog, stored.QuickItems[:], schema.EquipmentSlotQuickItem)
	if err != nil {
		return GetCharacterLoadoutResult{}, fmt.Errorf("quick items: %w", err)
	}
	result.Pouch, err = resolveOwnedLoadoutGroup(
		gameCatalog, stored.Pouch[:], schema.EquipmentSlotPouch)
	if err != nil {
		return GetCharacterLoadoutResult{}, fmt.Errorf("pouch: %w", err)
	}
	result.Physick, err = resolvePhysickGroup(gameCatalog, stored.Physick)
	if err != nil {
		return GetCharacterLoadoutResult{}, err
	}
	result.Spells, result.UsedMemorySlots, err = resolveSpellGroup(gameCatalog, stored.Spells)
	if err != nil {
		return GetCharacterLoadoutResult{}, err
	}
	if result.UsedMemorySlots > result.AvailableMemorySlots {
		return GetCharacterLoadoutResult{}, fmt.Errorf(
			"equipped spells use %d memory slots, but character %d has %d available",
			result.UsedMemorySlots, characterID, result.AvailableMemorySlots)
	}
	return result, nil
}

const loadoutAllOnes = uint32(0xFFFFFFFF)

func emptyCharacterLoadout(stored saveengine.CharacterLoadoutSnapshot) GetCharacterLoadoutResult {
	return GetCharacterLoadoutResult{
		SaveSessionID:         stored.SaveSessionID,
		SaveRevision:          stored.SaveRevision,
		CharacterID:           stored.CharacterID,
		Active:                stored.Active,
		RightHand:             []LoadoutSlot{},
		LeftHand:              []LoadoutSlot{},
		Arrows:                []LoadoutSlot{},
		Bolts:                 []LoadoutSlot{},
		Armor:                 []LoadoutSlot{},
		Talismans:             []LoadoutSlot{},
		QuickItems:            []LoadoutOwnedSlot{},
		Pouch:                 []LoadoutOwnedSlot{},
		ActiveQuickItem:       stored.ActiveQuickItem,
		Physick:               []LoadoutSlot{},
		Spells:                []LoadoutSpellSlot{},
		ActiveSpellIndex:      stored.ActiveSpellIndex,
		AvailableMemorySlots:  stored.AvailableMemorySlots,
		MemoryStones:          stored.MemoryStones,
		UnlockedTalismanSlots: stored.UnlockedTalismanSlots,
	}
}

type loadoutPosition struct {
	index int
	slot  schema.EquipmentSlot
}

func resolveLoadoutGroup(
	gameCatalog *gamecatalog.Catalog,
	stored saveengine.CharacterLoadoutSnapshot,
	positions []loadoutPosition,
	family schema.ItemFamily,
) ([]LoadoutSlot, error) {
	result := make([]LoadoutSlot, len(positions))
	for outputIndex, position := range positions {
		value := stored.Equipment[position.index]
		result[outputIndex] = LoadoutSlot{
			SlotType: position.slot,
			State:    LoadoutSlotEmpty,
			RawValue: value,
		}
		if saveengine.IsTechnicalEmptyEquipmentSlot(position.index, value) {
			continue
		}
		resource, name, iconPath, err := resolveLoadoutResource(gameCatalog, value, family)
		if err != nil {
			return nil, fmt.Errorf("slot %d: %w", outputIndex, err)
		}
		result[outputIndex].State = LoadoutSlotOccupied
		result[outputIndex].OwnedItemID = stored.EquipmentOwned[position.index]
		result[outputIndex].Resource = &resource
		result[outputIndex].Name = name
		result[outputIndex].IconPath = iconPath
	}
	return result, nil
}

func resolveTalismanGroup(
	gameCatalog *gamecatalog.Catalog,
	stored saveengine.CharacterLoadoutSnapshot,
) ([]LoadoutSlot, error) {
	result := make([]LoadoutSlot, 4)
	for index := range result {
		if index >= stored.UnlockedTalismanSlots {
			result[index] = LoadoutSlot{SlotType: schema.EquipmentSlotTalisman, State: LoadoutSlotLocked}
			continue
		}
		value := stored.Equipment[17+index]
		result[index] = LoadoutSlot{
			SlotType: schema.EquipmentSlotTalisman,
			State:    LoadoutSlotEmpty,
			RawValue: value,
		}
		if saveengine.IsTechnicalEmptyEquipmentSlot(17+index, value) {
			continue
		}
		resource, name, iconPath, err := resolveLoadoutResource(
			gameCatalog, value, schema.ItemFamilyTalisman)
		if err != nil {
			return nil, fmt.Errorf("talisman slot %d: %w", index, err)
		}
		result[index].State = LoadoutSlotOccupied
		result[index].OwnedItemID = stored.EquipmentOwned[17+index]
		result[index].Resource = &resource
		result[index].Name = name
		result[index].IconPath = iconPath
	}
	return result, nil
}

func resolveOwnedLoadoutGroup(
	gameCatalog *gamecatalog.Catalog,
	stored []saveengine.CharacterLoadoutOwnedItem,
	slotType schema.EquipmentSlot,
) ([]LoadoutOwnedSlot, error) {
	result := make([]LoadoutOwnedSlot, len(stored))
	for index, item := range stored {
		result[index] = LoadoutOwnedSlot{SlotType: slotType, State: LoadoutSlotEmpty}
		if item.OwnedItemID == "" {
			continue
		}
		resource, name, iconPath, err := resolveLoadoutResource(
			gameCatalog, item.GameID, schema.ItemFamilyGoods, schema.ItemFamilySpiritAsh)
		if err != nil {
			return nil, fmt.Errorf("slot %d: %w", index, err)
		}
		result[index] = LoadoutOwnedSlot{
			SlotType:    slotType,
			State:       LoadoutSlotOccupied,
			OwnedItemID: item.OwnedItemID,
			Resource:    &resource,
			Name:        name,
			IconPath:    iconPath,
			Quantity:    item.Quantity,
		}
	}
	return result, nil
}

func resolvePhysickGroup(
	gameCatalog *gamecatalog.Catalog,
	raw [2]uint32,
) ([]LoadoutSlot, error) {
	result := make([]LoadoutSlot, len(raw))
	for index, value := range raw {
		result[index] = LoadoutSlot{
			SlotType: schema.EquipmentSlotPhysick,
			State:    LoadoutSlotEmpty,
			RawValue: value,
		}
		if value == saveengine.PhysickEmptyTearID {
			continue
		}
		resource, name, iconPath, err := resolveLoadoutResource(
			gameCatalog, value, schema.ItemFamilyGoods)
		if err != nil {
			return nil, fmt.Errorf("physick slot %d: %w", index, err)
		}
		result[index].State = LoadoutSlotOccupied
		result[index].Resource = &resource
		result[index].Name = name
		result[index].IconPath = iconPath
	}
	return result, nil
}

func resolveSpellGroup(
	gameCatalog *gamecatalog.Catalog,
	raw [12]uint32,
) ([]LoadoutSpellSlot, int, error) {
	result := make([]LoadoutSpellSlot, len(raw))
	used := 0
	for index, value := range raw {
		result[index] = LoadoutSpellSlot{State: LoadoutSlotEmpty}
		if value == loadoutAllOnes {
			continue
		}
		if value == 0 || value >= gamecatalog.EquippedSpellRawIDLimit {
			return nil, 0, fmt.Errorf("spell slot %d: 0x%08X is not a raw MagicParam ID", index, value)
		}
		gameID := gamecatalog.EquippedSpellGameIDPrefix | value
		resource, found := gameCatalog.ItemByGameID(gameID)
		if !found || resource.Item == nil {
			return nil, 0, fmt.Errorf("spell slot %d: game ID 0x%08X is not a known item", index, gameID)
		}
		item := resource.Item
		if !item.Family.Known || item.Family.Value != schema.ItemFamilySpell {
			return nil, 0, fmt.Errorf("spell slot %d: game ID 0x%08X is not a spell", index, gameID)
		}
		if item.Spell == nil || !item.Spell.MemorySlots.Known {
			return nil, 0, fmt.Errorf("spell slot %d: spell 0x%08X has no known memory slots", index, gameID)
		}
		ref := resource.Ref()
		name, iconPath := itemPresentation(item)
		cost := int(item.Spell.MemorySlots.Value)
		used += cost
		result[index] = LoadoutSpellSlot{
			State:       LoadoutSlotOccupied,
			Resource:    &ref,
			Name:        name,
			IconPath:    iconPath,
			MemorySlots: cost,
		}
	}
	return result, used, nil
}

func resolveLoadoutResource(
	gameCatalog *gamecatalog.Catalog,
	gameID uint32,
	allowedFamilies ...schema.ItemFamily,
) (schema.ResourceRef, string, string, error) {
	resource, found := gameCatalog.ItemByGameID(gameID)
	if !found || resource.Item == nil {
		return schema.ResourceRef{}, "", "", fmt.Errorf("game ID 0x%08X is not a known item", gameID)
	}
	if len(allowedFamilies) > 0 {
		allowed := false
		if resource.Item.Family.Known {
			for _, family := range allowedFamilies {
				if resource.Item.Family.Value == family {
					allowed = true
					break
				}
			}
		}
		if !allowed {
			return schema.ResourceRef{}, "", "", fmt.Errorf(
				"game ID 0x%08X has item family %q, want one of %v",
				gameID, resource.Item.Family.Value, allowedFamilies)
		}
	}
	name, iconPath := itemPresentation(resource.Item)
	return resource.Ref(), name, iconPath, nil
}

func itemPresentation(item *schema.ItemDocument) (string, string) {
	var name, iconPath string
	if item.Presentation.Name.Known {
		name = item.Presentation.Name.Value
	}
	if item.Presentation.IconPath.Known {
		iconPath = item.Presentation.IconPath.Value
	}
	return name, iconPath
}
