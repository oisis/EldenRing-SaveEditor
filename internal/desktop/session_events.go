package desktop

import (
	"context"

	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// This file is the only place where a SaveEngine event becomes a Wails
// emission. SaveEngine publishes session.changed to a sink and knows nothing
// about the host; the bridge owns the host and turns one into the other.

// eventEmitter is the host call that delivers one event to the frontend. It
// exists so the bridge can be exercised without a real window: production
// leaves it nil and the bridge uses the Wails runtime.
type eventEmitter func(ctx context.Context, name string, data ...any)

// publishSessionChanged forwards one committed session event to the frontend.
//
// It is called by SaveEngine outside the engine lock, on the goroutine of the
// mutation that committed. Emission before the host has started is dropped
// rather than buffered: the frontend re-reads the current session and its
// eventSequence when its listener starts, so a dropped early event costs a
// resynchronisation and never a wrong cached view.
func (b *Bridge) publishSessionChanged(event saveengine.SessionChangedEvent) {
	ctx := b.hostContextOrNil()
	if ctx == nil {
		return
	}
	emit := b.emitEvent
	if emit == nil {
		emit = runtime.EventsEmit
	}
	emit(ctx, saveengine.SessionChangedEventName, event)
}
