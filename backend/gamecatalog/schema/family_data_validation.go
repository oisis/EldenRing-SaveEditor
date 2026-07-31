package schema

import "fmt"

func validateArmorData(data ArmorData, sources map[SourceID]struct{}) error {
	return validateFactValidators([]struct {
		name string
		fact anyFactValidator
	}{
		{"item.armor.sourceRowID", factValidator(data.SourceRowID)},
		{"item.armor.iconIDMale", factValidator(data.IconIDMale)},
		{"item.armor.iconIDFemale", factValidator(data.IconIDFemale)},
		{"item.armor.sortID", factValidator(data.SortID)},
		{"item.armor.sortGroupID", factValidator(data.SortGroupID)},
		{"item.armor.weight", factValidator(data.Weight)},
		{"item.armor.physical", factValidator(data.Physical)},
		{"item.armor.strike", factValidator(data.Strike)},
		{"item.armor.slash", factValidator(data.Slash)},
		{"item.armor.pierce", factValidator(data.Pierce)},
		{"item.armor.magic", factValidator(data.Magic)},
		{"item.armor.fire", factValidator(data.Fire)},
		{"item.armor.lightning", factValidator(data.Lightning)},
		{"item.armor.holy", factValidator(data.Holy)},
		{"item.armor.immunity", factValidator(data.Immunity)},
		{"item.armor.robustness", factValidator(data.Robustness)},
		{"item.armor.focus", factValidator(data.Focus)},
		{"item.armor.vitality", factValidator(data.Vitality)},
		{"item.armor.poise", factValidator(data.Poise)},
		{"item.armor.headEquipable", factValidator(data.HeadEquipable)},
		{"item.armor.bodyEquipable", factValidator(data.BodyEquipable)},
		{"item.armor.armEquipable", factValidator(data.ArmEquipable)},
		{"item.armor.legEquipable", factValidator(data.LegEquipable)},
	}, sources)
}

func validateTalismanData(data TalismanData, sources map[SourceID]struct{}) error {
	return validateFactValidators([]struct {
		name string
		fact anyFactValidator
	}{
		{"item.talisman.sourceRowID", factValidator(data.SourceRowID)},
		{"item.talisman.iconID", factValidator(data.IconID)},
		{"item.talisman.sortID", factValidator(data.SortID)},
		{"item.talisman.sortGroupID", factValidator(data.SortGroupID)},
		{"item.talisman.weight", factValidator(data.Weight)},
	}, sources)
}

func validateSpellData(data SpellData, sources map[SourceID]struct{}) error {
	return validateFactValidators([]struct {
		name string
		fact anyFactValidator
	}{
		{"item.spell.sourceRowID", factValidator(data.SourceRowID)},
		{"item.spell.iconID", factValidator(data.IconID)},
		{"item.spell.sortID", factValidator(data.SortID)},
		{"item.spell.fpCost", factValidator(data.FPCost)},
		{"item.spell.staminaCost", factValidator(data.StaminaCost)},
		{"item.spell.memorySlots", factValidator(data.MemorySlots)},
		{"item.spell.requiredIntelligence", factValidator(data.RequiredIntelligence)},
		{"item.spell.requiredFaith", factValidator(data.RequiredFaith)},
		{"item.spell.requiredArcane", factValidator(data.RequiredArcane)},
	}, sources)
}

func validateSpiritAshData(data SpiritAshData, sources map[SourceID]struct{}) error {
	return validateFactValidators([]struct {
		name string
		fact anyFactValidator
	}{
		{"item.spiritAsh.sourceRowID", factValidator(data.SourceRowID)},
		{"item.spiritAsh.iconID", factValidator(data.IconID)},
		{"item.spiritAsh.sortID", factValidator(data.SortID)},
		{"item.spiritAsh.sortGroupID", factValidator(data.SortGroupID)},
		{"item.spiritAsh.reinforceGoodsID", factValidator(data.ReinforceGoodsID)},
		{"item.spiritAsh.reinforceMaterialID", factValidator(data.ReinforceMaterialID)},
		{"item.spiritAsh.reinforcePrice", factValidator(data.ReinforcePrice)},
	}, sources)
}

func validateGoodsData(data GoodsData, sources map[SourceID]struct{}) error {
	return validateFactValidators([]struct {
		name string
		fact anyFactValidator
	}{
		{"item.goods.sourceRowID", factValidator(data.SourceRowID)},
		{"item.goods.iconID", factValidator(data.IconID)},
		{"item.goods.sortID", factValidator(data.SortID)},
		{"item.goods.sortGroupID", factValidator(data.SortGroupID)},
		{"item.goods.goodsType", factValidator(data.GoodsType)},
		{"item.goods.weight", factValidator(data.Weight)},
		{"item.goods.maxQuantity", factValidator(data.MaxQuantity)},
		{"item.goods.maxRepository", factValidator(data.MaxRepository)},
		{"item.goods.tutorialFlagID", factValidator(data.TutorialFlagID)},
		{"item.goods.isEquipable", factValidator(data.IsEquipable)},
		{"item.goods.isConsumable", factValidator(data.IsConsumable)},
		{"item.goods.isDiscardable", factValidator(data.IsDiscardable)},
		{"item.goods.isDepositable", factValidator(data.IsDepositable)},
		{"item.goods.isDroppable", factValidator(data.IsDroppable)},
	}, sources)
}

func validateGestureData(data GestureData, sources map[SourceID]struct{}) error {
	if err := validateFact("item.gesture.goodsSourceRowID", data.GoodsSourceRowID, sources); err != nil {
		return err
	}
	if err := validateFact("item.gesture.iconID", data.IconID, sources); err != nil {
		return err
	}
	if len(data.Slots) == 0 {
		return fmt.Errorf("item.gesture.slots must not be empty")
	}
	seenSlots := make(map[uint32]struct{}, len(data.Slots))
	for index, slot := range data.Slots {
		name := fmt.Sprintf("item.gesture.slots[%d]", index)
		if err := validateFact(name+".slotID", slot.SlotID, sources); err != nil {
			return err
		}
		if !slot.SlotID.Known {
			return fmt.Errorf("%s.slotID must be known", name)
		}
		if _, exists := seenSlots[slot.SlotID.Value]; exists {
			return fmt.Errorf("%s: duplicate slot ID %d", name, slot.SlotID.Value)
		}
		seenSlots[slot.SlotID.Value] = struct{}{}
		if err := validateFact(name+".itemID", slot.ItemID, sources); err != nil {
			return err
		}
		if !slot.ItemID.Known || slot.ItemID.Value == 0 {
			return fmt.Errorf("%s.itemID must be known and greater than zero", name)
		}
		if err := validateFact(name+".name", slot.Name, sources); err != nil {
			return err
		}
		if !slot.Name.Known || slot.Name.Value == "" {
			return fmt.Errorf("%s.name must be known and non-empty", name)
		}
		if err := validateFact(name+".category", slot.Category, sources); err != nil {
			return err
		}
		if !slot.Category.Known || slot.Category.Value == "" {
			return fmt.Errorf("%s.category must be known and non-empty", name)
		}
		if err := validateOptionalStringList(name+".flags", slot.Flags, sources); err != nil {
			return err
		}
		if err := validateParameterRecords(name+".sourceRecords", slot.SourceRecords, sources); err != nil {
			return err
		}
	}
	return nil
}
