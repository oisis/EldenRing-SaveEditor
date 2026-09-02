package desktop_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/apperror"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
	"github.com/oisis/EldenRing-SaveForge/internal/desktop"
)

// startedBridge wires a bridge with the supplied chooser and hands it the host
// context the way the application lifecycle does.
func startedBridge(chooser desktop.SaveFileChooser, catalog *gamecatalog.Catalog) *desktop.Bridge {
	bridge := desktop.NewBridge("dev", saveengine.New(), catalog, chooser)
	bridge.Startup(context.Background())
	return bridge
}

// TestSelectSaveFileReturnsTheHostPathUnchanged proves the bridge normalises
// nothing. The paths deliberately carry spaces, mixed case, a relative segment,
// an unusual extension and no extension at all: recognising a container belongs
// to SaveEngine, so none of these may be rewritten or refused here.
func TestSelectSaveFileReturnsTheHostPathUnchanged(t *testing.T) {
	for _, chosen := range []string{
		"/Users/tarnished/Elden Ring/ER0000.sl2",
		"  /leading and trailing spaces/ER0000.sl2  ",
		"/Users/Tarnished/MIXED Case/ER0000.SL2",
		"../relative/ER0000.sl2",
		"/backups/ER0000.sl2.20260824202530_bak",
		"/ps4/USER_DATA",
	} {
		t.Run(chosen, func(t *testing.T) {
			bridge := startedBridge(func(context.Context) (string, error) {
				return chosen, nil
			}, nil)

			path, err := bridge.SelectSaveFile()
			if err != nil {
				t.Fatalf("SelectSaveFile: %v", err)
			}
			if path != chosen {
				t.Errorf("SelectSaveFile = %q, want the host path unchanged %q", path, chosen)
			}
		})
	}
}

// TestSelectSaveFileTreatsCancellationAsAnOrdinaryOutcome pins the contract a
// cancelled dialog has: an empty path, no error, and above all no session. The
// bridge must not turn a cancellation into a load, a fallback path or a failure.
func TestSelectSaveFileTreatsCancellationAsAnOrdinaryOutcome(t *testing.T) {
	engine := saveengine.New()
	calls := 0
	bridge := desktop.NewBridge("dev", engine, nil, func(context.Context) (string, error) {
		calls++
		return "", nil
	})
	bridge.Startup(context.Background())

	path, err := bridge.SelectSaveFile()
	if err != nil {
		t.Fatalf("a cancelled dialog returned an error: %v", err)
	}
	if path != "" {
		t.Errorf("a cancelled dialog returned %q, want an empty path", path)
	}
	if calls != 1 {
		t.Errorf("the chooser was called %d times, want exactly 1", calls)
	}
	// Nothing was loaded: the identifier a cancelled dialog could only have
	// produced does not resolve, and neither does any other.
	if _, err := engine.GetSessionInfo(""); err == nil {
		t.Error("a cancelled dialog created a session")
	}
}

// TestSelectSaveFilePropagatesTheDialogFailureUnchanged proves a real dialog
// failure stays a failure and is never softened into a cancellation.
func TestSelectSaveFilePropagatesTheDialogFailureUnchanged(t *testing.T) {
	failure := errors.New("the host refused to open a dialog")
	bridge := startedBridge(func(context.Context) (string, error) {
		return "/ignored/ER0000.sl2", failure
	}, nil)

	path, err := bridge.SelectSaveFile()
	// The failure crosses the bridge as the shared error model: Wails carries
	// only a string, so the dialog error itself stays in the backend log and the
	// caller receives a classified, safe envelope instead.
	if err == nil {
		t.Fatal("SelectSaveFile = nil error, want the dialog failure")
	}
	public, decoded := desktop.DecodeBridgeError(err.Error())
	if !decoded || public.Code != apperror.CodeOperationFailed {
		t.Fatalf("SelectSaveFile error = %v, want an operation_failed envelope for %v",
			err, failure)
	}
	if strings.Contains(err.Error(), failure.Error()) {
		t.Errorf("the envelope leaks the raw dialog failure: %v", err)
	}
	if path != "" {
		t.Errorf("a failed dialog returned the path %q, want an empty path", path)
	}
}

// TestSelectSaveFileRefusesAnUnwiredHost covers the two wiring defects that must
// fail loudly instead of silently pretending the user cancelled: no chooser at
// all, and a chooser that has not received the host context yet.
func TestSelectSaveFileRefusesAnUnwiredHost(t *testing.T) {
	t.Run("no chooser", func(t *testing.T) {
		bridge := desktop.NewBridge("dev", saveengine.New(), nil, nil)
		bridge.Startup(context.Background())

		if _, err := bridge.SelectSaveFile(); err == nil {
			t.Fatal("SelectSaveFile succeeded without a chooser")
		}
	})

	t.Run("not started", func(t *testing.T) {
		called := false
		bridge := desktop.NewBridge("dev", saveengine.New(), nil, func(context.Context) (string, error) {
			called = true
			return "/ER0000.sl2", nil
		})

		if _, err := bridge.SelectSaveFile(); err == nil {
			t.Fatal("SelectSaveFile succeeded before Startup supplied the host context")
		}
		if called {
			t.Error("the chooser ran without a host context")
		}
	})
}

// TestStartupSuppliesTheHostContextToTheChooser proves the context reaches the
// dialog through the ordinary lifecycle rather than through a package-level
// variable, and that two bridges keep their own host.
func TestStartupSuppliesTheHostContextToTheChooser(t *testing.T) {
	type contextKey struct{}

	newBridge := func(want string) *desktop.Bridge {
		bridge := desktop.NewBridge("dev", saveengine.New(), nil, func(ctx context.Context) (string, error) {
			value, _ := ctx.Value(contextKey{}).(string)
			return value, nil
		})
		bridge.Startup(context.WithValue(context.Background(), contextKey{}, want))
		return bridge
	}

	first, second := newBridge("first-host"), newBridge("second-host")
	for _, testCase := range []struct {
		bridge *desktop.Bridge
		want   string
	}{{first, "first-host"}, {second, "second-host"}} {
		got, err := testCase.bridge.SelectSaveFile()
		if err != nil {
			t.Fatalf("SelectSaveFile: %v", err)
		}
		if got != testCase.want {
			t.Errorf("the chooser saw host %q, want %q", got, testCase.want)
		}
	}
}

// TestSelectSaveFileIsSafeUnderConcurrentStartup guards the only mutable state
// the bridge owns. It exists for the race detector: Startup and SelectSaveFile
// can be reached from different goroutines of the host.
func TestSelectSaveFileIsSafeUnderConcurrentStartup(t *testing.T) {
	bridge := desktop.NewBridge("dev", saveengine.New(), nil, func(context.Context) (string, error) {
		return "/ER0000.sl2", nil
	})

	var waiting sync.WaitGroup
	for range 8 {
		waiting.Add(2)
		go func() {
			defer waiting.Done()
			bridge.Startup(context.Background())
		}()
		go func() {
			defer waiting.Done()
			// Either outcome is legitimate: before Startup the call is refused,
			// after it the chooser answers. Neither may race.
			_, _ = bridge.SelectSaveFile()
		}()
	}
	waiting.Wait()
}
