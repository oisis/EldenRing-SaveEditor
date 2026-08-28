package application

import (
	"testing"
)

func TestGetStartingClasses_IncludesRegulation117Classes(t *testing.T) {
	app := &App{}
	classes := app.GetStartingClasses()

	if len(classes) != 12 {
		t.Fatalf("GetStartingClasses() returned %d classes, want 12", len(classes))
	}
	for i, cs := range classes {
		if int(cs.ID) != i {
			t.Fatalf("class at index %d has ID %d, want sorted by ID", i, cs.ID)
		}
	}
	if classes[10].Name != "Idus Knight" {
		t.Errorf("class 10 name = %q, want %q", classes[10].Name, "Idus Knight")
	}
	if classes[11].Name != "Heavy Knight" {
		t.Errorf("class 11 name = %q, want %q", classes[11].Name, "Heavy Knight")
	}
}
