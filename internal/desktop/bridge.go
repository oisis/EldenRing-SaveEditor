// Package desktop holds the Wails host bridge of SaveForge 2.0.
//
// The bridge is the only place where a public desktop method is declared. It
// owns no domain logic: every method delegates to a public backend endpoint and
// returns the endpoint result unchanged. The application root wires the bridge
// and stays a bootstrap and composition root only.
package desktop

import (
	"github.com/oisis/EldenRing-SaveForge/backend/endpoints/application"
)

// Bridge is the struct bound to Wails. Its exported methods are the desktop
// bridge surface reachable from the frontend.
type Bridge struct {
	// applicationVersion is injected by the composition root, which owns the
	// single source of the release version. The bridge neither reads a build
	// file nor defines a version constant of its own.
	applicationVersion string
}

// NewBridge builds the bridge with the application version supplied by its
// caller. An empty version is not rejected here: the endpoint owns that
// validation and the bridge must not duplicate it.
func NewBridge(applicationVersion string) *Bridge {
	return &Bridge{applicationVersion: applicationVersion}
}

// GetApplicationInfo delegates to the GetApplicationInfo endpoint and returns
// its result and error unchanged. It declares no capability, no schema version
// and no fallback version of its own.
func (b *Bridge) GetApplicationInfo() (application.GetApplicationInfoResult, error) {
	return application.GetApplicationInfo(b.applicationVersion)
}
