package migration

import (
	"fmt"
	"math"
	"strconv"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func buildSwordArtsNameFacts(
	regulation *RegulationData,
	gameText *GameTextData,
	legacy []swordArtsNameSeed,
) (map[int32]schema.Fact[string], error) {
	table, exists := regulation.Table(RegulationTableSwordArts)
	if !exists {
		return nil, fmt.Errorf("regulation table %q is not loaded", RegulationTableSwordArts)
	}
	legacyByID := make(map[int32]string, len(legacy))
	for _, value := range legacy {
		legacyByID[value.ID] = value.Name
	}

	result := make(map[int32]schema.Fact[string], table.RowCount())
	for _, row := range table.Rows() {
		if row.RowID > math.MaxInt32 {
			return nil, fmt.Errorf(
				"SwordArtsParam Row ID %d exceeds signed 32-bit range",
				row.RowID,
			)
		}
		textID, err := regulationInt32(row, "textId")
		if err != nil {
			return nil, err
		}
		legacyName := legacyByID[int32(row.RowID)]
		gameName, hasGameName := gameText.lookupName(textID)
		if hasGameName {
			method := "resolved SwordArtsParam row " +
				decimalRowID(row.RowID) +
				" textId " +
				strconv.FormatInt(int64(textID), 10) +
				" from the English game-text FMG extract"
			if legacyName != "" && legacyName != gameName.text {
				method += "; replaced conflicting legacy name " + strconv.Quote(legacyName)
			}
			result[int32(row.RowID)] = schema.Fact[string]{
				Known: true,
				Value: gameName.text,
				Provenance: schema.Provenance{
					Source: gameName.source,
					Method: method,
				},
			}
			continue
		}
		if legacyName != "" {
			result[int32(row.RowID)] = knownLegacyFact(
				legacyName,
				"FMG has no nonblank text for SwordArtsParam row "+
					decimalRowID(row.RowID)+
					"; copied from legacy SwordArtsNames",
			)
			continue
		}
		result[int32(row.RowID)] = unknownCatalogFact[string](
			"no nonblank FMG text or legacy SwordArtsNames entry for SwordArtsParam row " +
				decimalRowID(row.RowID),
		)
	}
	return result, nil
}

func (context *generationContext) swordArtsNameFact(id int32) schema.Fact[string] {
	name, exists := context.swordArtsNames[id]
	if !exists {
		return unknownCatalogFact[string](
			"SwordArtsParam row is unavailable for this swordArtsParamId",
		)
	}
	return name
}
