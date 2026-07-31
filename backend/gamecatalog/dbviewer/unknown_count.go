package dbviewer

import "reflect"

func countUnknownFacts(value any) int {
	return countUnknownValue(reflect.ValueOf(value))
}

func countUnknownValue(value reflect.Value) int {
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
			if value.Type().Field(index).PkgPath != "" {
				continue
			}
			count += countUnknownValue(value.Field(index))
		}
	case reflect.Array, reflect.Slice:
		for index := 0; index < value.Len(); index++ {
			count += countUnknownValue(value.Index(index))
		}
	}
	return count
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
