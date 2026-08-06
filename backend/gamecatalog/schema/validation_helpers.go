package schema

import (
	"fmt"
	"reflect"
)

func validateFact[T any](name string, fact Fact[T], sources map[SourceID]struct{}) error {
	return validateProvenance(name, fact.Provenance, sources)
}

// validateOptionalFact preserves compatibility with the two schema-v1
// prototype documents. New documents must either omit the fact completely or
// provide a fully formed fact with provenance, including for unknown values.
func validateOptionalFact[T any](name string, fact Fact[T], sources map[SourceID]struct{}) error {
	if isOmittedFact(fact) {
		return nil
	}
	return validateFact(name, fact, sources)
}

func isOmittedFact[T any](fact Fact[T]) bool {
	return !fact.Known &&
		reflect.ValueOf(fact.Value).IsZero() &&
		fact.Provenance.Source == "" &&
		fact.Provenance.Method == ""
}

func isNotApplicableFact[T any](fact Fact[T]) bool {
	return !fact.Known &&
		reflect.ValueOf(fact.Value).IsZero() &&
		fact.Provenance.MarksNotApplicable()
}

func validateProvenance(name string, provenance Provenance, sources map[SourceID]struct{}) error {
	if provenance.Source == "" {
		return fmt.Errorf("%s: provenance source is required", name)
	}
	if _, exists := sources[provenance.Source]; !exists {
		return fmt.Errorf("%s: unknown provenance source %q", name, provenance.Source)
	}
	if provenance.Method == "" {
		return fmt.Errorf("%s: provenance method is required", name)
	}
	hasExactReference := provenance.Table != "" || provenance.Row != "" || provenance.Field != ""
	if hasExactReference &&
		(provenance.Table == "" || provenance.Row == "" || provenance.Field == "") {
		return fmt.Errorf("%s: provenance table, row, and field must be provided together", name)
	}
	return nil
}
