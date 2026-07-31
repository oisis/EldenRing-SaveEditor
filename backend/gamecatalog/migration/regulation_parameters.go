package migration

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"strings"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

const sourceRegulationEquipParamGemRaw schema.SourceID = "regulation_equip_param_gem_raw"

const (
	equipParamGemFilename       = "EquipParamGem.param"
	equipParamGemRowCountOffset = 0x0A
	equipParamGemRowTableOffset = 0x40
	equipParamGemRowEntrySize   = 0x18
	equipParamGemMaskOffset     = 0x38
	equipParamGemMaskSize       = 8
	equipParamGemMaskBits       = 44
)

type RegulationParameterData struct {
	gemCompatibilityMasks map[uint32]uint64
	source                schema.DataSource
}

func ReadRegulationParameterDirectory(
	root string,
) (*RegulationParameterData, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("regulation parameter directory is required")
	}
	return readRegulationParameterFS(os.DirFS(root))
}

func readRegulationParameterFS(
	source fs.FS,
) (*RegulationParameterData, error) {
	if source == nil {
		return nil, errors.New("regulation parameter filesystem is required")
	}
	raw, err := fs.ReadFile(source, equipParamGemFilename)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", equipParamGemFilename, err)
	}
	masks, err := parseEquipParamGemMasks(raw)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", equipParamGemFilename, err)
	}
	sum := sha256.Sum256(raw)
	return &RegulationParameterData{
		gemCompatibilityMasks: masks,
		source: schema.DataSource{
			ID:       sourceRegulationEquipParamGemRaw,
			Kind:     "regulation_parameter",
			Location: "regulation.bin/params/EquipParamGem.param",
			Version:  hex.EncodeToString(sum[:]),
			Evidence: schema.EvidenceRegulation,
			Reviewed: true,
		},
	}, nil
}

func parseEquipParamGemMasks(raw []byte) (map[uint32]uint64, error) {
	if len(raw) < equipParamGemRowTableOffset {
		return nil, fmt.Errorf("truncated PARAM header")
	}
	rowCount := int(binary.LittleEndian.Uint16(
		raw[equipParamGemRowCountOffset : equipParamGemRowCountOffset+2],
	))
	tableEnd := equipParamGemRowTableOffset + rowCount*equipParamGemRowEntrySize
	if rowCount == 0 || tableEnd > len(raw) {
		return nil, fmt.Errorf("invalid row table")
	}
	result := make(map[uint32]uint64, rowCount)
	for index := 0; index < rowCount; index++ {
		entry := equipParamGemRowTableOffset + index*equipParamGemRowEntrySize
		rawRowID := binary.LittleEndian.Uint64(raw[entry : entry+8])
		if rawRowID > math.MaxUint32 {
			return nil, fmt.Errorf("row %d ID %d exceeds uint32", index, rawRowID)
		}
		rowID := uint32(rawRowID)
		if _, duplicate := result[rowID]; duplicate {
			return nil, fmt.Errorf("duplicate Row ID %d", rowID)
		}
		dataOffset64 := binary.LittleEndian.Uint64(raw[entry+8 : entry+16])
		if dataOffset64 > uint64(len(raw)) {
			return nil, fmt.Errorf("Row ID %d data offset exceeds file", rowID)
		}
		dataOffset := int(dataOffset64)
		maskEnd := dataOffset + equipParamGemMaskOffset + equipParamGemMaskSize
		if dataOffset < tableEnd || maskEnd > len(raw) {
			return nil, fmt.Errorf("Row ID %d has invalid data range", rowID)
		}
		mask := binary.LittleEndian.Uint64(
			raw[dataOffset+equipParamGemMaskOffset : maskEnd],
		)
		if mask>>equipParamGemMaskBits != 0 {
			return nil, fmt.Errorf(
				"Row ID %d sets compatibility bits above %d",
				rowID,
				equipParamGemMaskBits-1,
			)
		}
		result[rowID] = mask
	}
	return result, nil
}

func (data *RegulationParameterData) gemCompatibilityMask(
	rowID uint32,
) (uint64, bool) {
	if data == nil {
		return 0, false
	}
	mask, exists := data.gemCompatibilityMasks[rowID]
	return mask, exists
}

func (data *RegulationParameterData) manifestSource() schema.DataSource {
	return data.source
}
