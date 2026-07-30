package schema

import "fmt"

func validateFact[T any](name string, fact Fact[T], sources map[SourceID]struct{}) error {
	return validateProvenance(name, fact.Provenance, sources)
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
	return nil
}
