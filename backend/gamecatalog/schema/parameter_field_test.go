package schema_test

import (
	"reflect"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestParameterFieldContainsOnlyTheReferencedFieldName(t *testing.T) {
	fieldType := reflect.TypeOf(schema.ParameterField{})
	if fieldType.NumField() != 1 || fieldType.Field(0).Name != "Name" {
		t.Fatalf("ParameterField runtime fields = %v, want only Name", fieldType)
	}
}
