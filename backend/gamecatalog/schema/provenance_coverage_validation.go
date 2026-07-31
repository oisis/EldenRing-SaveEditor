package schema

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

func validateRegulationProvenanceCoverage(
	name string,
	value any,
	records []ParameterRecord,
) error {
	fieldsByRecord := make(map[string]map[string]struct{}, len(records))
	for _, record := range records {
		key := provenanceRecordKey(
			record.Provenance.Source,
			record.Table,
			strconv.FormatInt(record.RowID, 10),
		)
		if fieldsByRecord[key] == nil {
			fieldsByRecord[key] = make(map[string]struct{}, len(record.Fields))
		}
		for _, field := range record.Fields {
			fieldsByRecord[key][field.Name] = struct{}{}
		}
	}
	return validateValueProvenanceCoverage(
		reflect.ValueOf(value),
		name,
		fieldsByRecord,
	)
}

func validateValueProvenanceCoverage(
	value reflect.Value,
	path string,
	fieldsByRecord map[string]map[string]struct{},
) error {
	if !value.IsValid() {
		return nil
	}
	for value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}
	if value.Type() == reflect.TypeOf(Provenance{}) {
		provenance := value.Interface().(Provenance)
		if provenance.Table == "" {
			return nil
		}
		key := provenanceRecordKey(
			provenance.Source,
			provenance.Table,
			provenance.Row,
		)
		fields, exists := fieldsByRecord[key]
		if !exists {
			return fmt.Errorf(
				"%s: Regulation provenance %s row %s is not covered by sourceRecords",
				path,
				provenance.Table,
				provenance.Row,
			)
		}
		for _, field := range strings.Split(provenance.Field, ",") {
			field = strings.TrimSpace(field)
			if _, exists := fields[field]; !exists {
				return fmt.Errorf(
					"%s: Regulation provenance %s row %s field %q is not covered by sourceRecords",
					path,
					provenance.Table,
					provenance.Row,
					field,
				)
			}
		}
		return nil
	}
	switch value.Kind() {
	case reflect.Struct:
		for index := 0; index < value.NumField(); index++ {
			field := value.Type().Field(index)
			if field.PkgPath != "" {
				continue
			}
			if err := validateValueProvenanceCoverage(
				value.Field(index),
				path+"."+field.Name,
				fieldsByRecord,
			); err != nil {
				return err
			}
		}
	case reflect.Slice, reflect.Array:
		for index := 0; index < value.Len(); index++ {
			if err := validateValueProvenanceCoverage(
				value.Index(index),
				fmt.Sprintf("%s[%d]", path, index),
				fieldsByRecord,
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func provenanceRecordKey(source SourceID, table, row string) string {
	return string(source) + "\x00" + table + "\x00" + row
}
