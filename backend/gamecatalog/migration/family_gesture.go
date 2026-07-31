package migration

import (
	"fmt"
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
		slotID := knownLegacyFact(slot.SlotID, "copied from legacy AllGestures save slot ID")
		itemID := knownLegacyFact(slot.ItemID, "copied from legacy AllGestures item ID")
		var sourceRecords []schema.ParameterRecord
		rows := context.gestureRows[slot.ItemID&0x0FFFFFFF]
		if len(rows) > 0 {
			matchingRows := make([]ParameterRow, 0, 1)
			for _, row := range rows {
				if row.RowID*2+1 == slot.SlotID {
					matchingRows = append(matchingRows, row)
				}
			}
			if len(matchingRows) != 1 {
				return nil, fmt.Errorf(
					"legacy gesture slot %d for item 0x%08X has %d matching GestureParam rows",
					slot.SlotID,
					slot.ItemID,
					len(matchingRows),
				)
			}
			row := matchingRows[0]
			slotID = knownRegulationDerivedFact(
				row.RowID*2+1,
				RegulationTableGesture,
				"derived canonical save slot ID as GestureParam Row ID * 2 + 1",
				row.RowID,
				"Row ID",
			)
			itemID = knownRegulationDerivedFact(
				uint32(0x40000000)|(slot.ItemID&0x0FFFFFFF),
				RegulationTableGesture,
				"derived full goods item ID from GestureParam itemId",
				row.RowID,
				"itemId",
			)
			sourceRecords = parameterRecordsForRows(
				RegulationTableGesture,
				matchingRows,
				context.regulation,
				"itemId",
			)
		}
		data.Slots[index] = schema.GestureSlotRecord{
			SlotID:        slotID,
			ItemID:        itemID,
			Name:          knownLegacyFact(slot.Name, "copied from legacy AllGestures name"),
			Category:      knownLegacyFact(slot.Category, "copied from legacy AllGestures category"),
			Flags:         knownLegacyFact(cloneStrings(slot.Flags), "copied from legacy AllGestures flags"),
			SourceRecords: sourceRecords,
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
