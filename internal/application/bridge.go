package application

import "context"

// Startup exposes the Wails lifecycle hook to the package-main facade without
// adding another bound method to App.
func Startup(app *App, ctx context.Context) {
	app.startup(ctx)
}

// Shutdown exposes the Wails lifecycle hook to the package-main facade without
// adding another bound method to App.
func Shutdown(app *App, ctx context.Context) {
	app.shutdown(ctx)
}

// SetDiagnosticJournal attaches the session journal created by package main.
func SetDiagnosticJournal(app *App, journal *DiagnosticJournal) {
	app.journal = journal
}

// NewWailsJournalLogger creates the existing Wails logger adapter.
func NewWailsJournalLogger(journal *DiagnosticJournal) *wailsJournalLogger {
	return newWailsJournalLogger(journal)
}
