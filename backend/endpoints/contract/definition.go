package contract

import (
	"fmt"
	"strings"
)

// Kind identifies whether an endpoint is read-only or may change application
// or save state.
type Kind string

const (
	Getter   Kind = "getter"
	Mutation Kind = "mutation"
)

// Definition is the implementation-independent contract of one public
// SaveForge 2.0 endpoint. It deliberately does not define a transport request
// or runtime handler before the corresponding GameCatalog and SaveEngine
// contracts exist.
type Definition struct {
	Name                       string
	ID                         string
	Kind                       Kind
	SupportedResourceTypes     string
	SupportedResourceVariables []string
	Description                string
}

// MustDefine validates a static endpoint definition during package
// initialization and returns it unchanged.
func MustDefine(definition Definition) Definition {
	if err := definition.Validate(); err != nil {
		panic(err)
	}
	return definition
}

// Validate rejects incomplete definitions and unstable endpoint identifiers.
func (definition Definition) Validate() error {
	if definition.Name == "" {
		return fmt.Errorf("endpoint name is required")
	}
	if !validEndpointID(definition.ID) {
		return fmt.Errorf("endpoint %s has invalid EndpointID %q", definition.Name, definition.ID)
	}
	if definition.Kind != Getter && definition.Kind != Mutation {
		return fmt.Errorf("endpoint %s has invalid kind %q", definition.Name, definition.Kind)
	}
	if strings.TrimSpace(definition.SupportedResourceTypes) == "" {
		return fmt.Errorf("endpoint %s must declare supported resource types", definition.Name)
	}
	if strings.TrimSpace(definition.Description) == "" {
		return fmt.Errorf("endpoint %s description is required", definition.Name)
	}

	seen := make(map[string]struct{}, len(definition.SupportedResourceVariables))
	for _, variable := range definition.SupportedResourceVariables {
		if variable == "" || strings.TrimSpace(variable) != variable {
			return fmt.Errorf("endpoint %s has invalid supported variable %q", definition.Name, variable)
		}
		if _, exists := seen[variable]; exists {
			return fmt.Errorf("endpoint %s repeats supported variable %q", definition.Name, variable)
		}
		seen[variable] = struct{}{}
	}

	return nil
}

func validEndpointID(value string) bool {
	if value == "" || value[0] == '_' || value[len(value)-1] == '_' || strings.Contains(value, "__") {
		return false
	}
	for _, character := range value {
		if (character < 'a' || character > 'z') &&
			(character < '0' || character > '9') && character != '_' {
			return false
		}
	}
	return true
}
