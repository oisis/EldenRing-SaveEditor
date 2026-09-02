package saveengine

import (
	"fmt"

	"github.com/oisis/EldenRing-SaveForge/backend/apperror"
)

// This file resolves one opaque OwnedItemID back to the single physical record
// it was minted for. It owns no container layout: it selects the container the
// token was minted in and reads exactly that one through the shared reader of
// that container, so there is one source of truth for anchors, bounds, section
// sizes, sentinels, the quantity mask, physical indexes and minting.

// OwnedItem is one physical owned record addressed by its opaque identity. It
// carries the raw physical values of the record and no catalog identity at all:
// resolving the handle to an ItemDocument belongs to the endpoint above.
//
// SaveRevision is the revision the record was read under, and the one
// OwnedItemID is valid for. OwnedItemID is echoed back exactly as supplied: it
// is never parsed, trimmed, normalised or reconstructed.
//
// Container is the physical container the record lives in, "inventory" or
// "storage". A record in one container never shares an identity with the record
// at the same ContainerSection/PhysicalIndex in the other, and this getter never
// searches the other container as a fallback.
type OwnedItem struct {
	SaveSessionID    string `json:"saveSessionID"`
	SaveRevision     string `json:"saveRevision"`
	OwnedItemID      string `json:"ownedItemID"`
	CharacterID      int    `json:"characterID"`
	Container        string `json:"container"`
	ContainerSection string `json:"containerSection"`
	PhysicalIndex    int    `json:"physicalIndex"`
	GaItemHandle     uint32 `json:"gaItemHandle"`
	Quantity         uint32 `json:"quantity"`
	AcquisitionIndex uint32 `json:"acquisitionIndex"`
}

// GetOwnedItem returns the one physical record ownedItemID was minted for. Like
// the other character readers it reads the session's private snapshot through
// the codec only: it opens no file, writes nothing and changes no session. It
// calls no other getter and no endpoint.
//
// saveSessionID is matched exactly. It is never trimmed, normalised or guessed,
// so an empty, unknown or already closed identifier is rejected instead of
// resolving to a session.
//
// ownedItemID is opaque and revision-scoped, not a stable item reference: it is
// valid only inside the session that minted it and only while that session's
// revision is unchanged. It is compared byte for byte and never parsed. An
// empty, unknown, fabricated, foreign or retired token is rejected; a token
// minted for another character is rejected with its own error, because the
// remedy differs from "this token does not exist".
//
// The whole operation runs under Engine.mutex, so the revision the result
// reports is the revision the record was read under and the one the token was
// validated against.
//
// Only the container the token was minted in is read. A token whose physical
// record is gone, or whose record no longer carries that exact token, is a hard
// error: there is no fallback position, no second candidate record, no search of
// the other container and no zero-value success.
func (engine *Engine) GetOwnedItem(
	saveSessionID string,
	characterID int,
	ownedItemID string,
) (OwnedItem, error) {
	if saveSessionID == "" {
		return OwnedItem{}, apperror.MissingField("saveSessionID")
	}

	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	loaded, exists := engine.sessions[saveSessionID]
	if !exists {
		return OwnedItem{}, apperror.UnknownSaveSession(saveSessionID)
	}
	if characterID < 0 || characterID >= characterSlotCount {
		return OwnedItem{}, fmt.Errorf("characterID %d is outside the range 0..%d",
			characterID, characterSlotCount-1)
	}

	locator, err := loaded.session.resolveOwnedItemID(characterID, ownedItemID)
	if err != nil {
		return OwnedItem{}, err
	}

	// The physical coordinates come from the locator, which the matched record
	// has to equal exactly, so only the stored values still have to be copied out
	// of the record itself.
	item := OwnedItem{
		SaveSessionID:    saveSessionID,
		SaveRevision:     loaded.session.revisionString(),
		OwnedItemID:      ownedItemID,
		CharacterID:      characterID,
		Container:        locator.container,
		ContainerSection: locator.containerSection,
		PhysicalIndex:    locator.physicalIndex,
	}

	// ponytail: the two containers keep separate record types by contract, so the
	// two branches stay literal instead of sharing a generic finder over one
	// invented common type.
	switch locator.container {
	case ownedContainerInventory:
		records, err := readInventoryRecords(loaded, characterID)
		if err != nil {
			return OwnedItem{}, err
		}
		for _, record := range records {
			if record.ContainerSection != locator.containerSection ||
				record.PhysicalIndex != locator.physicalIndex || record.OwnedItemID != ownedItemID {
				continue
			}
			item.GaItemHandle = record.GaItemHandle
			item.Quantity = record.Quantity
			item.AcquisitionIndex = record.AcquisitionIndex
			return item, nil
		}
	case ownedContainerStorage:
		records, err := readStorageRecords(loaded, characterID)
		if err != nil {
			return OwnedItem{}, err
		}
		for _, record := range records {
			if record.ContainerSection != locator.containerSection ||
				record.PhysicalIndex != locator.physicalIndex || record.OwnedItemID != ownedItemID {
				continue
			}
			item.GaItemHandle = record.GaItemHandle
			item.Quantity = record.Quantity
			item.AcquisitionIndex = record.AcquisitionIndex
			return item, nil
		}
	}
	return OwnedItem{}, fmt.Errorf(
		"ownedItemID %q no longer addresses a record of character %d", ownedItemID, characterID)
}
