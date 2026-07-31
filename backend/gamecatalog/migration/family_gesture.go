package migration

import (
	"sort"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func (context *generationContext) buildGestureData(
	item seed,
	goodsRow ParameterRow,
	hasGoodsRow bool,
) (*schema.GestureData, error) {
	data := &schema.GestureData{}
	if hasGoodsRow {
		iconID, err := regulationUint32(goodsRow, "iconId")
		if err != nil {
			return nil, err
		}
		data.GoodsSourceRowID = knownRegulationFact(
			goodsRow.RowID,
			RegulationTableGoods,
			"Row ID",
			goodsRow.RowID,
		)
		data.IconID = knownRegulationFact(
			iconID,
			RegulationTableGoods,
			"iconId",
			goodsRow.RowID,
		)
	} else {
		data.GoodsSourceRowID = unknownLegacyFact[uint32](
			"gesture item has no EquipParamGoods row",
		)
		data.IconID = unknownLegacyFact[uint32](
			"gesture item has no EquipParamGoods iconId",
		)
	}

	slots := append([]gestureSlotSeed(nil), context.gesturesByItem[item.ID]...)
	sort.Slice(slots, func(i, j int) bool {
		return slots[i].SlotID < slots[j].SlotID
	})
	data.Slots = make([]schema.GestureSlotRecord, len(slots))
	for index, slot := range slots {
		data.Slots[index] = schema.GestureSlotRecord{
			SlotID:        knownLegacyFact(slot.SlotID, "copied from legacy AllGestures save slot ID"),
			ItemID:        knownLegacyFact(slot.ItemID, "copied from legacy AllGestures item ID"),
			Name:          knownLegacyFact(slot.Name, "copied from legacy AllGestures name"),
			Category:      knownLegacyFact(slot.Category, "copied from legacy AllGestures category"),
			Flags:         knownLegacyFact(cloneStrings(slot.Flags), "copied from legacy AllGestures flags"),
			SourceRecords: nil,
		}
	}
	return data, nil
}

func buildSlotOnlyGestureGroups(snapshot legacySnapshot) []legacyItemGroup {
	knownItems := make(map[uint32]struct{}, len(snapshot.Items))
	for _, item := range snapshot.Items {
		knownItems[item.ID] = struct{}{}
	}
	slotOnly := make(map[uint32][]gestureSlotSeed)
	for _, slot := range snapshot.GestureSlots {
		if _, exists := knownItems[slot.ItemID]; exists {
			continue
		}
		slotOnly[slot.ItemID] = append(slotOnly[slot.ItemID], slot)
	}
	ids := make([]uint32, 0, len(slotOnly))
	for id := range slotOnly {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	groups := make([]legacyItemGroup, 0, len(ids))
	for _, id := range ids {
		slots := slotOnly[id]
		flags := mergeGestureFlags(nil, slots)
		groups = append(groups, legacyItemGroup{
			Canonical: seed{
				ID:            id,
				HasLegacyItem: false,
				Category:      "gestures",
				Name:          slots[0].Name,
				Flags:         flags,
			},
		})
	}
	return groups
}
