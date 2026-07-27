package main

import (
	"context"

	"github.com/oisis/EldenRing-SaveForge/internal/application"
)

// App preserves the stable Wails go.main.App binding while the implementation
// lives in the internal application package.
type App struct {
	*application.App
}

func NewApp() *App {
	return &App{App: application.NewApp()}
}

func (a *App) startup(ctx context.Context) {
	application.Startup(a.App, ctx)
}

func (a *App) shutdown(ctx context.Context) {
	application.Shutdown(a.App, ctx)
}

func (a *App) setDiagnosticJournal(journal *application.DiagnosticJournal) {
	application.SetDiagnosticJournal(a.App, journal)
}
