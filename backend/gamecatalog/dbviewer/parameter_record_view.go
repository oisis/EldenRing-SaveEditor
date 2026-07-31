package dbviewer

import (
	"fmt"
	"reflect"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

var parameterRecordType = reflect.TypeOf(schema.ParameterRecord{})

type parameterRecordView struct {
	Scope          string
	Table          string
	RowID          int64
	Fields         []schema.ParameterField
	Source         schema.SourceID
	Method         string
	SourceLocation string
}

func (server *Server) parameterRecordViews(item *schema.ItemDocument) []parameterRecordView {
	if item == nil {
		return nil
	}
	records := server.parameterRecords("Item", item.SourceRecords)
	value := reflect.ValueOf(item).Elem()
	for index := 0; index < value.NumField(); index++ {
		fieldType := value.Type().Field(index)
		if fieldType.PkgPath != "" || fieldType.Name == "SourceRecords" {
			continue
		}
		scope := joinFactLabel("Item", fieldLabel(fieldType))
		records = append(records, server.collectParameterRecords(value.Field(index), scope)...)
	}
	return records
}

func (server *Server) collectParameterRecords(value reflect.Value, scope string) []parameterRecordView {
	if !value.IsValid() {
		return nil
	}
	for value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}

	if value.Kind() == reflect.Slice && value.Type().Elem() == parameterRecordType {
		records := make([]schema.ParameterRecord, value.Len())
		for index := 0; index < value.Len(); index++ {
			records[index] = value.Index(index).Interface().(schema.ParameterRecord)
		}
		return server.parameterRecords(scope, records)
	}

	var records []parameterRecordView
	switch value.Kind() {
	case reflect.Struct:
		for index := 0; index < value.NumField(); index++ {
			fieldType := value.Type().Field(index)
			if fieldType.PkgPath != "" {
				continue
			}
			fieldScope := scope
			if fieldType.Name != "SourceRecords" {
				fieldScope = joinFactLabel(scope, fieldLabel(fieldType))
			}
			records = append(records, server.collectParameterRecords(value.Field(index), fieldScope)...)
		}
	case reflect.Array, reflect.Slice:
		for index := 0; index < value.Len(); index++ {
			elementScope := fmt.Sprintf("%s %d", scope, index+1)
			records = append(records, server.collectParameterRecords(value.Index(index), elementScope)...)
		}
	}
	return records
}

func (server *Server) parameterRecords(scope string, records []schema.ParameterRecord) []parameterRecordView {
	views := make([]parameterRecordView, 0, len(records))
	for _, record := range records {
		source := server.sources[record.Provenance.Source]
		views = append(views, parameterRecordView{
			Scope:          scope,
			Table:          record.Table,
			RowID:          record.RowID,
			Fields:         append([]schema.ParameterField(nil), record.Fields...),
			Source:         record.Provenance.Source,
			Method:         record.Provenance.Method,
			SourceLocation: source.Location,
		})
	}
	return views
}
