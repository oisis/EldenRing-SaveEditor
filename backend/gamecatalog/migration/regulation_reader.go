package migration

import (
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strconv"
	"strings"
)

const regulationProvenanceRoot = "regulation.bin/csv/"

// ReadRegulationCSVDirectory loads the supported parameter tables from csvDirectory.
func ReadRegulationCSVDirectory(csvDirectory string) (*RegulationData, error) {
	if strings.TrimSpace(csvDirectory) == "" {
		return nil, errors.New("regulation CSV directory is required")
	}
	return readRegulationFS(os.DirFS(csvDirectory))
}

func readRegulationFS(source fs.FS) (*RegulationData, error) {
	if source == nil {
		return nil, errors.New("regulation CSV filesystem is required")
	}

	data := &RegulationData{
		tables: make(map[RegulationTableName]*RegulationTable, len(regulationTableSpecs)),
	}
	for _, spec := range regulationTableSpecs {
		raw, err := fs.ReadFile(source, spec.filename)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", spec.filename, err)
		}
		table, err := parseRegulationTable(spec, raw)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", spec.filename, err)
		}
		data.tables[spec.name] = table
	}
	return data, nil
}

func parseRegulationTable(spec regulationTableSpec, raw []byte) (*RegulationTable, error) {
	reader := csv.NewReader(bytes.NewReader(raw))
	reader.Comma = ';'
	reader.ReuseRecord = false
	reader.FieldsPerRecord = -1

	header, err := reader.Read()
	if errors.Is(err, io.EOF) {
		return nil, errors.New("missing header")
	}
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	if len(header) == 0 {
		return nil, errors.New("missing header")
	}
	header[0] = strings.TrimPrefix(header[0], "\uFEFF")

	rowIDColumn, err := validateRegulationHeader(header)
	if err != nil {
		return nil, err
	}
	reader.FieldsPerRecord = len(header)

	sum := sha256.Sum256(raw)
	table := &RegulationTable{
		name: spec.name,
		source: RegulationSource{
			Location: regulationProvenanceRoot + spec.filename,
			Version:  hex.EncodeToString(sum[:]),
		},
		rowsByID: make(map[uint32]ParameterRow),
	}

	for recordNumber := 2; ; recordNumber++ {
		record, readErr := reader.Read()
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("invalid row %d: %w", recordNumber, readErr)
		}

		rowID, parseErr := parseRawRowID(record[rowIDColumn])
		if parseErr != nil {
			return nil, fmt.Errorf("invalid row %d: %w", recordNumber, parseErr)
		}
		if _, duplicate := table.rowsByID[rowID]; duplicate {
			return nil, fmt.Errorf("duplicate Row ID %d at row %d", rowID, recordNumber)
		}

		fields := make([]ParameterField, len(header))
		for index, name := range header {
			fields[index] = ParameterField{Name: name, RawValue: record[index]}
		}
		table.rowIDs = append(table.rowIDs, rowID)
		table.rowsByID[rowID] = ParameterRow{RowID: rowID, Fields: fields}
	}

	return table, nil
}

func validateRegulationHeader(header []string) (int, error) {
	rowIDColumn := -1
	seen := make(map[string]struct{}, len(header))
	for index, name := range header {
		if name == "" {
			return -1, fmt.Errorf("header column %d has no name", index+1)
		}
		if _, duplicate := seen[name]; duplicate {
			return -1, fmt.Errorf("duplicate header column %q", name)
		}
		seen[name] = struct{}{}
		if name == "Row ID" {
			rowIDColumn = index
		}
	}
	if rowIDColumn == -1 {
		return -1, errors.New(`missing required "Row ID" header`)
	}
	return rowIDColumn, nil
}

func parseRawRowID(raw string) (uint32, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, errors.New("Row ID is empty")
	}
	rowID, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("Row ID %q is not an unsigned 32-bit decimal integer", raw)
	}
	return uint32(rowID), nil
}
