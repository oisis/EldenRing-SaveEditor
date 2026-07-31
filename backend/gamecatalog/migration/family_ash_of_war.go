package migration

import (
	"fmt"
	"strings"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func (context *generationContext) buildAshOfWarData(
	item seed,
	row ParameterRow,
) (*schema.AshOfWarData, error) {
	if item.AoWCompatMask == nil {
		return nil, fmt.Errorf("legacy full-width AoW compatibility mask is missing")
	}
	rawMask, exists := context.regulationParams.gemCompatibilityMask(row.RowID)
	if !exists {
		return nil, fmt.Errorf(
			"EquipParamGem.param has no Row ID %d",
			row.RowID,
		)
	}
	if *item.AoWCompatMask != rawMask {
		return nil, fmt.Errorf(
			"row %d full AoW compatibility mismatch: legacy 0x%X, Regulation PARAM 0x%X",
			row.RowID,
			*item.AoWCompatMask,
			rawMask,
		)
	}
	if err := crossCheckAoWCompatibilityMask(rawMask, row); err != nil {
		return nil, err
	}
	iconID, err := regulationUint32(row, "iconId")
	if err != nil {
		return nil, err
	}
	sortID, err := regulationUint32(row, "sortId")
	if err != nil {
		return nil, err
	}
	sortGroupID, err := regulationUint8(row, "sortGroupId")
	if err != nil {
		return nil, err
	}
	swordArtsParamID, err := regulationInt32(row, "swordArtsParamId")
	if err != nil {
		return nil, err
	}
	defaultAffinity, err := regulationUint8(row, "defaultWepAttr")
	if err != nil {
		return nil, err
	}
	return &schema.AshOfWarData{
		SourceRowID:      knownRegulationFact(row.RowID, RegulationTableGem, "Row ID", row.RowID),
		IconID:           knownRegulationFact(iconID, RegulationTableGem, "iconId", row.RowID),
		SortID:           knownRegulationFact(sortID, RegulationTableGem, "sortId", row.RowID),
		SortGroupID:      knownRegulationFact(sortGroupID, RegulationTableGem, "sortGroupId", row.RowID),
		SwordArtsParamID: knownRegulationFact(swordArtsParamID, RegulationTableGem, "swordArtsParamId", row.RowID),
		SwordArtsName:    context.swordArtsNameFact(swordArtsParamID),
		DefaultAffinity:  knownRegulationFact(defaultAffinity, RegulationTableGem, "defaultWepAttr", row.RowID),
		CompatibilityMask: schema.Fact[uint64]{
			Known: true,
			Value: rawMask,
			Provenance: schema.Provenance{
				Source: sourceRegulationEquipParamGemRaw,
				Method: "parsed the full 44-bit compatibility field and cross-checked its lower 40 bits against EquipParamGem.csv",
				Table:  string(RegulationTableGem),
				Row:    decimalRowID(row.RowID),
				Field:  "canMountWep[0:44]",
			},
		},
		CompatibleClassNames: knownLegacyFact(cloneStrings(item.AoWCompatibleClasses), "derived from legacy AoWCompatMasks and CanMountWepNames"),
	}, nil
}

func crossCheckAoWCompatibilityMask(rawMask uint64, row ParameterRow) error {
	var csvMask uint64
	var bit uint
	for _, field := range row.Fields {
		if !strings.HasPrefix(field.Name, "canMountWep_") {
			continue
		}
		if bit >= 64 {
			return fmt.Errorf("row %d has more than 64 canMountWep fields", row.RowID)
		}
		enabled, err := regulationBool(row, field.Name)
		if err != nil {
			return err
		}
		if enabled {
			csvMask |= uint64(1) << bit
		}
		bit++
	}
	if bit != 36 {
		return fmt.Errorf(
			"row %d has %d canMountWep fields, want 36",
			row.RowID,
			bit,
		)
	}
	reserved, err := regulationUint8(row, "reserved_canMountWep")
	if err != nil {
		return err
	}
	if reserved > 0x0F {
		return fmt.Errorf(
			"row %d reserved_canMountWep = %d exceeds four bits",
			row.RowID,
			reserved,
		)
	}
	csvMask |= uint64(reserved) << 36
	const csvMaskBits = 40
	const availableMask = (uint64(1) << csvMaskBits) - 1
	if rawMask&availableMask != csvMask {
		return fmt.Errorf(
			"row %d AoW compatibility mismatch: PARAM lower mask 0x%X, CSV 0x%X",
			row.RowID,
			rawMask&availableMask,
			csvMask,
		)
	}
	return nil
}
