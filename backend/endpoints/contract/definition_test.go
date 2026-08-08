package contract

import (
	"strings"
	"testing"
)

func TestDefinitionValidate(t *testing.T) {
	t.Parallel()

	valid := Definition{
		Name:                       "SetGestureUnlocked",
		ID:                         "set_gesture_unlocked",
		Kind:                       Mutation,
		SupportedResourceTypes:     "ItemDocument: Gesture",
		SupportedResourceVariables: []string{"characterID", "gestureKind", "gestureKey", "unlocked"},
		Description:                "Sets one gesture unlock state.",
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid definition rejected: %v", err)
	}

	tests := []struct {
		name        string
		mutate      func(*Definition)
		wantInError string
	}{
		{name: "missing name", mutate: func(value *Definition) { value.Name = "" }, wantInError: "name"},
		{name: "invalid id", mutate: func(value *Definition) { value.ID = "SetGestureUnlocked" }, wantInError: "EndpointID"},
		{name: "invalid kind", mutate: func(value *Definition) { value.Kind = "command" }, wantInError: "kind"},
		{name: "missing resources", mutate: func(value *Definition) { value.SupportedResourceTypes = "" }, wantInError: "resource types"},
		{name: "duplicate variable", mutate: func(value *Definition) {
			value.SupportedResourceVariables = append(value.SupportedResourceVariables, "characterID")
		}, wantInError: "repeats"},
		{name: "missing description", mutate: func(value *Definition) { value.Description = "" }, wantInError: "description"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			definition := valid
			definition.SupportedResourceVariables = append([]string(nil), valid.SupportedResourceVariables...)
			test.mutate(&definition)
			err := definition.Validate()
			if err == nil || !strings.Contains(err.Error(), test.wantInError) {
				t.Fatalf("Validate() error = %v, want containing %q", err, test.wantInError)
			}
		})
	}
}
