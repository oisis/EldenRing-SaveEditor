package schema

type RelatedEventFlagKind string

const (
	RelatedEventFlagWhetblade RelatedEventFlagKind = "whetblade_related"
	RelatedEventFlagAoWMenu   RelatedEventFlagKind = "aow_menu_unlock"
)

type RelatedItemKind string

const (
	RelatedItemBundledAcquisition RelatedItemKind = "bundled_acquisition"
)

type ItemLinks struct {
	AboutTutorialID   Fact[uint32]         `json:"aboutTutorialID"`
	RelatedEventFlags []RelatedEventFlag   `json:"relatedEventFlags"`
	RelatedItems      []RelatedItem        `json:"relatedItems"`
	MapFragment       *MapFragmentMetadata `json:"mapFragment"`
}

type RelatedEventFlag struct {
	Kind        Fact[RelatedEventFlagKind] `json:"kind"`
	EventFlagID Fact[uint32]               `json:"eventFlagID"`
}

type RelatedItem struct {
	Kind   Fact[RelatedItemKind] `json:"kind"`
	GameID Fact[uint32]          `json:"gameID"`
}

type MapFragmentMetadata struct {
	Name           Fact[string] `json:"name"`
	Area           Fact[string] `json:"area"`
	AcquiredFlagID Fact[uint32] `json:"acquiredFlagID"`
}
