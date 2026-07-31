package schema

import "fmt"

func validateParameterRecords(name string, records []ParameterRecord, sources map[SourceID]struct{}) error {
	for recordIndex, record := range records {
		recordName := fmt.Sprintf("%s[%d]", name, recordIndex)
		if record.Table == "" {
			return fmt.Errorf("%s.table is required", recordName)
		}
		if err := validateProvenance(recordName, record.Provenance, sources); err != nil {
			return err
		}
		if len(record.Fields) == 0 {
			return fmt.Errorf("%s.fields must identify at least one used source field", recordName)
		}
		seen := make(map[string]struct{}, len(record.Fields))
		for fieldIndex, field := range record.Fields {
			fieldName := fmt.Sprintf("%s.fields[%d]", recordName, fieldIndex)
			if field.Name == "" {
				return fmt.Errorf("%s.name is required", fieldName)
			}
			if _, exists := seen[field.Name]; exists {
				return fmt.Errorf("%s: duplicate field %q", recordName, field.Name)
			}
			seen[field.Name] = struct{}{}
		}
	}
	return nil
}
