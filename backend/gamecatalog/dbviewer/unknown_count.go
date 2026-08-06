package dbviewer

import (
	"reflect"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func countUnknownFacts(value any) int {
	return countUnknownValue(reflect.ValueOf(value), false)
}

func countUnknownFactsForFamily(value any, family schema.ItemFamily) int {
	return countUnknownValue(
		reflect.ValueOf(value),
		supportsNotApplicableMetadata(family),
	)
}

func countUnknownValue(value reflect.Value, allowNotApplicable bool) int {
	if !value.IsValid() {
		return 0
	}
	for value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return 0
		}
		value = value.Elem()
	}

	if known, factLike := knownField(value); factLike {
		if !known {
			return 1
		}
		return 0
	}

	count := 0
	switch value.Kind() {
	case reflect.Struct:
		for index := 0; index < value.NumField(); index++ {
			field := value.Type().Field(index)
			if field.PkgPath != "" ||
				(allowNotApplicable && isNotApplicableUnknownField(value, field, value.Field(index))) {
				continue
			}
			count += countUnknownValue(value.Field(index), allowNotApplicable)
		}
	case reflect.Array, reflect.Slice:
		for index := 0; index < value.Len(); index++ {
			count += countUnknownValue(value.Index(index), allowNotApplicable)
		}
	}
	return count
}

func isNotApplicableUnknownField(
	owner reflect.Value,
	field reflect.StructField,
	value reflect.Value,
) bool {
	switch field.Name {
	case "RequiredContainerID", "WhetbladeName":
		provenance, unknownFact := unknownFactProvenance(value)
		return unknownFact && provenance.MarksNotApplicable()
	case "Affinity":
		if owner.Type() != reflect.TypeOf(schema.ItemVariant{}) || !variantKindIsUpgrade(owner) {
			return false
		}
		provenance, unknownFact := unknownFactProvenance(value)
		return unknownFact && provenance.MarksNotApplicable()
	default:
		return false
	}
}

func variantKindIsUpgrade(variant reflect.Value) bool {
	kind := variant.FieldByName("Kind")
	known := kind.FieldByName("Known")
	value := kind.FieldByName("Value")
	return known.IsValid() && known.Bool() && value.IsValid() &&
		value.String() == string(schema.ItemVariantUpgrade)
}

// unknownFactProvenance returns the provenance of a fact that is unknown and
// carries a zero value, so callers can tell a declared exception from a fact
// that merely stayed unresolved.
func unknownFactProvenance(value reflect.Value) (schema.Provenance, bool) {
	for value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return schema.Provenance{}, false
		}
		value = value.Elem()
	}
	if known, factLike := knownField(value); !factLike || known {
		return schema.Provenance{}, false
	}
	rawValue := value.FieldByName("Value")
	if !rawValue.IsValid() || !rawValue.IsZero() {
		return schema.Provenance{}, false
	}
	provenance, ok := value.FieldByName("Provenance").Interface().(schema.Provenance)
	return provenance, ok
}

func knownField(value reflect.Value) (bool, bool) {
	if value.Kind() != reflect.Struct {
		return false, false
	}
	known := value.FieldByName("Known")
	provenance := value.FieldByName("Provenance")
	if !known.IsValid() || known.Kind() != reflect.Bool || !provenance.IsValid() {
		return false, false
	}
	return known.Bool(), true
}
