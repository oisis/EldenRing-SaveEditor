package schema

import "fmt"

func validateItemLinks(
	name string,
	links ItemLinks,
	ownerGameID uint32,
	unlocks []ItemUnlock,
	sources map[SourceID]struct{},
) error {
	if err := validateOptionalFact(name+".aboutTutorialID", links.AboutTutorialID, sources); err != nil {
		return err
	}
	if links.AboutTutorialID.Known && links.AboutTutorialID.Value == 0 {
		return fmt.Errorf("%s.aboutTutorialID must be greater than zero when known", name)
	}
	if err := validateOptionalNonEmptyString(name+".whetbladeName", links.WhetbladeName, sources); err != nil {
		return err
	}

	occupiedFlags := make(map[uint32]struct{}, len(unlocks)+len(links.RelatedEventFlags)+1)
	for _, unlock := range unlocks {
		if unlock.EventFlagID.Known {
			occupiedFlags[unlock.EventFlagID.Value] = struct{}{}
		}
	}
	for index, link := range links.RelatedEventFlags {
		linkName := fmt.Sprintf("%s.relatedEventFlags[%d]", name, index)
		if err := validateFact(linkName+".kind", link.Kind, sources); err != nil {
			return err
		}
		if !link.Kind.Known || !validRelatedEventFlagKind(link.Kind.Value) {
			return fmt.Errorf("%s.kind must be known and supported", linkName)
		}
		if err := validateFact(linkName+".eventFlagID", link.EventFlagID, sources); err != nil {
			return err
		}
		if !link.EventFlagID.Known || link.EventFlagID.Value == 0 {
			return fmt.Errorf("%s.eventFlagID must be known and greater than zero", linkName)
		}
		if _, exists := occupiedFlags[link.EventFlagID.Value]; exists {
			return fmt.Errorf("%s: duplicate event flag ID %d", linkName, link.EventFlagID.Value)
		}
		occupiedFlags[link.EventFlagID.Value] = struct{}{}
	}

	occupiedItems := make(map[uint32]struct{}, len(links.RelatedItems))
	for index, link := range links.RelatedItems {
		linkName := fmt.Sprintf("%s.relatedItems[%d]", name, index)
		if err := validateFact(linkName+".kind", link.Kind, sources); err != nil {
			return err
		}
		if !link.Kind.Known || link.Kind.Value != RelatedItemBundledAcquisition {
			return fmt.Errorf("%s.kind must be known and supported", linkName)
		}
		if err := validateFact(linkName+".gameID", link.GameID, sources); err != nil {
			return err
		}
		if !link.GameID.Known || link.GameID.Value == 0 {
			return fmt.Errorf("%s.gameID must be known and greater than zero", linkName)
		}
		if ownerGameID != 0 && link.GameID.Value == ownerGameID {
			return fmt.Errorf("%s cannot point to the owning item", linkName)
		}
		if _, exists := occupiedItems[link.GameID.Value]; exists {
			return fmt.Errorf("%s: duplicate related item ID 0x%08X", linkName, link.GameID.Value)
		}
		occupiedItems[link.GameID.Value] = struct{}{}
	}

	if links.MapFragment == nil {
		return nil
	}
	if err := validateFact(name+".mapFragment.name", links.MapFragment.Name, sources); err != nil {
		return err
	}
	if !links.MapFragment.Name.Known || links.MapFragment.Name.Value == "" {
		return fmt.Errorf("%s.mapFragment.name must be known and non-empty", name)
	}
	if err := validateFact(name+".mapFragment.area", links.MapFragment.Area, sources); err != nil {
		return err
	}
	if !links.MapFragment.Area.Known || links.MapFragment.Area.Value == "" {
		return fmt.Errorf("%s.mapFragment.area must be known and non-empty", name)
	}
	if err := validateFact(name+".mapFragment.acquiredFlagID", links.MapFragment.AcquiredFlagID, sources); err != nil {
		return err
	}
	if !links.MapFragment.AcquiredFlagID.Known || links.MapFragment.AcquiredFlagID.Value == 0 {
		return fmt.Errorf("%s.mapFragment.acquiredFlagID must be known and greater than zero", name)
	}
	if _, exists := occupiedFlags[links.MapFragment.AcquiredFlagID.Value]; exists {
		return fmt.Errorf(
			"%s.mapFragment: duplicate event flag ID %d",
			name,
			links.MapFragment.AcquiredFlagID.Value,
		)
	}
	return nil
}

func validRelatedEventFlagKind(kind RelatedEventFlagKind) bool {
	return kind == RelatedEventFlagWhetblade || kind == RelatedEventFlagAoWMenu
}
