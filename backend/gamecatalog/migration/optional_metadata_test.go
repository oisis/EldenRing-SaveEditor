package migration

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"
)

func TestAbsentOptionalMetadataIsOmitted(t *testing.T) {
	acquisition := buildAcquisition(seed{}, true)
	if acquisition.RequiredContainerID.Known || acquisition.RequiredContainerID.Provenance != (schema.Provenance{}) {
		t.Fatalf("absent required container = %#v, want omitted fact", acquisition.RequiredContainerID)
	}

	links := buildLinks(linksSeed{}, true)
	if links.WhetbladeName.Known || links.WhetbladeName.Provenance != (schema.Provenance{}) {
		t.Fatalf("absent whetblade name = %#v, want omitted fact", links.WhetbladeName)
	}
}
