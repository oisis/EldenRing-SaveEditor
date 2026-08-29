package catalog_test

import (
	"encoding/json"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/catalog"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

const (
	classKind = "class"
	// Ten base classes plus the Regulation 1.17 pair 10 Idus Knight and
	// 11 Heavy Knight.
	classCount = 12
)

func TestGetResourcesReturnsClassesAsNonItems(t *testing.T) {
	cat := newStoredCatalog(t)

	result, err := catalog.GetResources(cat, classKind, "", "", "", "samurai", 0, 0)
	if err != nil {
		t.Fatalf("GetResources(class, samurai): %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("samurai search total = %d, want 1", result.Total)
	}
	if len(result.Resources) != 1 || result.Resources[0].Name != "Samurai" || result.Resources[0].Key != "7" {
		t.Fatalf("class resource = %+v, want Samurai (key 7)", result.Resources)
	}

	all, err := catalog.GetResources(cat, classKind, "", "", "", "", 0, 0)
	if err != nil || all.Total != classCount {
		t.Fatalf("all classes total = %d, error %v", all.Total, err)
	}
}

func TestGetResourceReturnsIndependentClassDocument(t *testing.T) {
	cat := newStoredCatalog(t)

	result, err := catalog.GetResource(cat, classKind, "6")
	if err != nil {
		t.Fatalf("GetResource(class, 6): %v", err)
	}
	resource := result.Resource
	if resource.Kind != schema.ResourceKindClass || resource.Class == nil ||
		resource.Class.StartingClassID.Value != 6 ||
		resource.Class.Name.Value != "Confessor" ||
		!resource.Class.Level.Known || resource.Class.Level.Value != 10 {
		t.Fatalf("class resource = %+v", resource)
	}

	// The level reaches the transport as its own fact under the "level" key, so
	// a client can present the base Rune Level of a class without deriving it.
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal class resource: %v", err)
	}
	var decoded struct {
		Resource struct {
			Class struct {
				Level struct {
					Known      bool            `json:"known"`
					Value      uint32          `json:"value"`
					Provenance json.RawMessage `json:"provenance"`
				} `json:"level"`
			} `json:"class"`
		} `json:"resource"`
	}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("decode class resource: %v", err)
	}
	if !decoded.Resource.Class.Level.Known || decoded.Resource.Class.Level.Value != 10 {
		t.Errorf("serialised level = %+v, want the known value 10", decoded.Resource.Class.Level)
	}
	if len(decoded.Resource.Class.Level.Provenance) == 0 {
		t.Error("serialised level carries no provenance")
	}

	// Deep copy immutability check
	resource.Class.Name.Value = "mutated"
	resource.Class.Level.Value = 99
	again, err := catalog.GetResource(cat, classKind, "6")
	if err != nil || again.Resource.Class.Name.Value != "Confessor" ||
		again.Resource.Class.Level.Value != 10 {
		t.Fatalf("catalog class name = %q, level = %d, error %v",
			again.Resource.Class.Name.Value, again.Resource.Class.Level.Value, err)
	}
}
