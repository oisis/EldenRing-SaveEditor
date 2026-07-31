package migration

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

type generationContext struct {
	regulation       *RegulationData
	regulationParams *RegulationParameterData
	manifest         schema.Manifest
	aliasesByItem    map[uint32][]aliasSeed
	gesturesByItem   map[uint32][]gestureSlotSeed
	gestureRows      map[uint32][]ParameterRow
	swordArtsNames   map[int32]schema.Fact[string]
	technicalItems   map[uint32]technicalRecordSeed
}

func newGenerationContext(
	options GenerateOptions,
	sourceVersions migrationSourceVersions,
	snapshot legacySnapshot,
) (*generationContext, error) {
	gestureRows, err := indexGestureRows(options.Regulation)
	if err != nil {
		return nil, err
	}
	swordArtsNames, err := buildSwordArtsNameFacts(
		options.Regulation,
		options.GameText,
		snapshot.SwordArtsNames,
	)
	if err != nil {
		return nil, err
	}
	context := &generationContext{
		regulation:       options.Regulation,
		regulationParams: options.RegulationParams,
		manifest:         buildManifest(options, sourceVersions),
		aliasesByItem:    make(map[uint32][]aliasSeed),
		gesturesByItem:   make(map[uint32][]gestureSlotSeed),
		gestureRows:      gestureRows,
		swordArtsNames:   swordArtsNames,
		technicalItems:   make(map[uint32]technicalRecordSeed, len(snapshot.TechnicalRecords)),
	}
	for _, alias := range snapshot.Aliases {
		context.aliasesByItem[alias.CanonicalID] = append(
			context.aliasesByItem[alias.CanonicalID],
			alias,
		)
	}
	for _, slot := range snapshot.GestureSlots {
		context.gesturesByItem[slot.ItemID] = append(
			context.gesturesByItem[slot.ItemID],
			slot,
		)
	}
	for _, value := range snapshot.TechnicalRecords {
		context.technicalItems[value.ID] = value
	}
	return context, nil
}

func indexGestureRows(regulation *RegulationData) (map[uint32][]ParameterRow, error) {
	table, exists := regulation.Table(RegulationTableGesture)
	if !exists {
		return nil, fmt.Errorf("regulation table %q is not loaded", RegulationTableGesture)
	}
	result := make(map[uint32][]ParameterRow)
	for _, row := range table.Rows() {
		rawItemID, ok := row.Field("itemId")
		if !ok {
			return nil, fmt.Errorf("GestureParam row %d has no itemId field", row.RowID)
		}
		value, err := strconv.ParseInt(rawItemID, 10, 32)
		if err != nil || value < 0 {
			continue
		}
		result[uint32(value)] = append(result[uint32(value)], row)
	}
	for itemID := range result {
		sort.Slice(result[itemID], func(i, j int) bool {
			return result[itemID][i].RowID < result[itemID][j].RowID
		})
	}
	return result, nil
}
