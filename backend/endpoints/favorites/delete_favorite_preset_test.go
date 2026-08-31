package favorites

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

func TestDeleteFavoritePresetEndpointSuccess(t *testing.T) {
	path := writeEndpointFavoritesFixture(t, "pc", map[int]bool{2: true})
	engine := saveengine.New()
	session, err := engine.LoadSave(path, "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := DeleteFavoritePreset(engine, session.SaveSessionID, 2, "0")
	if err != nil {
		t.Fatalf("DeleteFavoritePreset: %v", err)
	}

	if result.SaveSessionID != session.SaveSessionID {
		t.Errorf("SaveSessionID = %q, want %q", result.SaveSessionID, session.SaveSessionID)
	}
	if result.SaveRevision != "1" {
		t.Errorf("SaveRevision = %q, want \"1\"", result.SaveRevision)
	}
	if result.FavoriteSlotID != 2 {
		t.Errorf("FavoriteSlotID = %d, want 2", result.FavoriteSlotID)
	}
}

func TestDeleteFavoritePresetEndpointRejectsMissingEngine(t *testing.T) {
	if _, err := DeleteFavoritePreset(nil, "session", 0, "0"); err == nil {
		t.Fatal("expected error for nil engine, got nil")
	}
}

func TestDeleteFavoritePresetEndpointDelegatesValidation(t *testing.T) {
	path := writeEndpointFavoritesFixture(t, "pc", nil)
	engine := saveengine.New()
	session, err := engine.LoadSave(path, "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	// Delegated validation check: slot out of range
	if _, err := DeleteFavoritePreset(engine, session.SaveSessionID, 15, "0"); err == nil {
		t.Fatal("expected error for out-of-range favoriteSlotID, got nil")
	}
}
