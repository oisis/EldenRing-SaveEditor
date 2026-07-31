package schema

import (
	"fmt"
	"reflect"
	"strings"
)

const saveForgeValueSuffix = "-sfv"

func validateSaveForgeValues(
	name string,
	value reflect.Value,
	sources map[SourceID]struct{},
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

	switch value.Kind() {
	case reflect.Struct:
		valueType := value.Type()
		for index := 0; index < value.NumField(); index++ {
			fieldType := valueType.Field(index)
			if fieldType.PkgPath != "" {
				continue
			}
			jsonName := strings.Split(fieldType.Tag.Get("json"), ",")[0]
			fieldName := fieldType.Name
			if jsonName != "" && jsonName != "-" {
				fieldName = jsonName
			}
			fieldPath := name + "." + fieldName
			if strings.HasSuffix(jsonName, saveForgeValueSuffix) {
				if err := validateSaveForgeValue(
					fieldPath,
					value,
					index,
					strings.TrimSuffix(jsonName, saveForgeValueSuffix),
					sources,
				); err != nil {
					return err
				}
				continue
			}
			if err := validateSaveForgeValues(
				fieldPath,
				value.Field(index),
				sources,
			); err != nil {
				return err
			}
		}
	case reflect.Array, reflect.Slice:
		for index := 0; index < value.Len(); index++ {
			if err := validateSaveForgeValues(
				fmt.Sprintf("%s[%d]", name, index),
				value.Index(index),
				sources,
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateSaveForgeValue(
	name string,
	parent reflect.Value,
	index int,
	authoritativeJSONName string,
	sources map[SourceID]struct{},
) error {
	legacyPointer := parent.Field(index)
	if legacyPointer.Kind() != reflect.Pointer {
		return fmt.Errorf("%s must be an optional pointer", name)
	}
	if legacyPointer.IsNil() {
		return nil
	}
	legacyFact := legacyPointer.Elem()
	legacyKnown, legacyValue, legacyProvenance, ok := reflectedFactParts(legacyFact)
	if !ok {
		return fmt.Errorf("%s must point to a Fact", name)
	}
	if !legacyKnown {
		return fmt.Errorf("%s must be known when present", name)
	}
	if err := validateProvenance(name, legacyProvenance, sources); err != nil {
		return err
	}
	if legacyProvenance.Source != SourceSaveForgeLegacy {
		return fmt.Errorf(
			"%s provenance source = %q, want %q",
			name,
			legacyProvenance.Source,
			SourceSaveForgeLegacy,
		)
	}

	authoritative, exists := fieldByJSONName(parent, authoritativeJSONName)
	if !exists {
		return fmt.Errorf(
			"%s has no authoritative sibling %q",
			name,
			authoritativeJSONName,
		)
	}
	authoritativeKnown, authoritativeValue, _, isFact := reflectedFactParts(authoritative)
	if isFact {
		if !authoritativeKnown {
			return fmt.Errorf("%s authoritative sibling must be known", name)
		}
	} else {
		authoritativeValue = authoritative
	}
	if legacyValue.Type() != authoritativeValue.Type() {
		return fmt.Errorf(
			"%s value type %s differs from authoritative sibling type %s",
			name,
			legacyValue.Type(),
			authoritativeValue.Type(),
		)
	}
	if reflect.DeepEqual(legacyValue.Interface(), authoritativeValue.Interface()) {
		return fmt.Errorf("%s duplicates the authoritative value", name)
	}
	return nil
}

func reflectedFactParts(
	value reflect.Value,
) (bool, reflect.Value, Provenance, bool) {
	for value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return false, reflect.Value{}, Provenance{}, false
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return false, reflect.Value{}, Provenance{}, false
	}
	known := value.FieldByName("Known")
	factValue := value.FieldByName("Value")
	provenance := value.FieldByName("Provenance")
	if !known.IsValid() || known.Kind() != reflect.Bool ||
		!factValue.IsValid() || !factValue.CanInterface() ||
		!provenance.IsValid() || !provenance.CanInterface() {
		return false, reflect.Value{}, Provenance{}, false
	}
	typedProvenance, ok := provenance.Interface().(Provenance)
	if !ok {
		return false, reflect.Value{}, Provenance{}, false
	}
	return known.Bool(), factValue, typedProvenance, true
}

func fieldByJSONName(value reflect.Value, name string) (reflect.Value, bool) {
	valueType := value.Type()
	for index := 0; index < value.NumField(); index++ {
		jsonName := strings.Split(valueType.Field(index).Tag.Get("json"), ",")[0]
		if jsonName == name {
			return value.Field(index), true
		}
	}
	return reflect.Value{}, false
}
