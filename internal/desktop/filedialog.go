package desktop

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// SaveFileChooser asks the host to let the user pick one existing save file and
// returns the path the host reported.
//
// Cancelling is an ordinary user outcome and not an error: a cancelled dialog
// returns an empty path and a nil error, which every caller must distinguish
// from a chosen path. An error means the dialog itself failed.
//
// The port exists so the desktop bridge can be exercised without a real window.
// It is deliberately a function and not an interface with one implementation:
// the host has exactly one dialog to offer at this stage.
type SaveFileChooser func(ctx context.Context) (string, error)

// SaveTargetChooser asks the host for an explicit output path. Cancelling is an
// empty path and nil error, exactly like opening; no file is created by the
// dialog itself.
type SaveTargetChooser func(ctx context.Context, suggestedName string) (string, error)

// saveFileFilters are the extensions this build offers as a convenience only.
// The "all files" entry stays available on purpose: a container is recognised
// from its leading magic by SaveEngine, never from a file name, so an unusual
// extension must not make a valid save unselectable.
//
// The list carries the PC containers and the confirmed PS4 one, ".dat". No
// further extension is added on a guess: an extension nobody has confirmed as a
// container would suggest support this build cannot promise, while costing
// nothing to leave out because the "all files" entry already reaches it.
var saveFileFilters = []runtime.FileFilter{
	{
		DisplayName: "Elden Ring saves (*.sl2, *.co2, *.bak, *.dat)",
		Pattern:     "*.sl2;*.co2;*.bak;*.dat",
	},
	{DisplayName: "All files (*.*)", Pattern: "*.*"},
}

// NewWailsSaveFileChooser is the production chooser. It opens the host's native
// single-file dialog for the supplied Wails context and returns the path
// unchanged: nothing here trims, resolves, recases or validates it, because
// recognising a save belongs to SaveEngine alone.
func NewWailsSaveFileChooser() SaveFileChooser {
	return func(ctx context.Context) (string, error) {
		return runtime.OpenFileDialog(ctx, runtime.OpenDialogOptions{
			Title:   "Open Save",
			Filters: saveFileFilters,
			// An alias must resolve to the file it points at, so the path handed
			// to SaveEngine names a real container rather than a link to one.
			ResolvesAliases: true,
		})
	}
}

func NewWailsSaveTargetChooser() SaveTargetChooser {
	return func(ctx context.Context, suggestedName string) (string, error) {
		return runtime.SaveFileDialog(ctx, runtime.SaveDialogOptions{
			Title:           "Save As",
			DefaultFilename: suggestedName,
			Filters:         saveFileFilters,
		})
	}
}
