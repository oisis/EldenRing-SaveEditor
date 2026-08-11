package dbviewer

import "github.com/oisis/EldenRing-SaveForge/backend/gamecatalog/schema"

func (server *Server) storageFacts(storage schema.ItemStorage) []factView {
	result := []factView{
		server.fact("Record mode", storage.RecordMode.Known, storage.RecordMode.Value, storage.RecordMode.Provenance),
		server.fact("Maximum inventory", storage.MaxInventory.Known, storage.MaxInventory.Value, storage.MaxInventory.Provenance),
	}
	result = appendOptionalStorageFact(
		server,
		result,
		"Safe Mode maximum inventory",
		storage.SafeModeMaxInventory,
	)
	result = appendOptionalStorageFact(
		server,
		result,
		"Maximum inventory — SaveForge value",
		storage.MaxInventorySFV,
	)
	result = append(
		result,
		server.fact("Maximum storage", storage.MaxStorage.Known, storage.MaxStorage.Value, storage.MaxStorage.Provenance),
	)
	result = appendOptionalStorageFact(
		server,
		result,
		"Safe Mode maximum storage",
		storage.SafeModeMaxStorage,
	)
	return appendOptionalStorageFact(
		server,
		result,
		"Maximum storage — SaveForge value",
		storage.MaxStorageSFV,
	)
}

func appendOptionalStorageFact(
	server *Server,
	facts []factView,
	label string,
	value *schema.Fact[uint32],
) []factView {
	if value == nil {
		return facts
	}
	return append(
		facts,
		server.fact(
			label,
			value.Known,
			value.Value,
			value.Provenance,
		),
	)
}
