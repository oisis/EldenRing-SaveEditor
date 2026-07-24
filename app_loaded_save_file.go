package main

// GetLoadedSaveFileName returns only the safe basename of the currently
// loaded save. It deliberately never exposes the local directory path.
func (a *App) GetLoadedSaveFileName() string {
	a.saveMu.RLock()
	defer a.saveMu.RUnlock()

	if a.save == nil {
		return ""
	}
	return safeSaveFileName(a.lastSavePath)
}
