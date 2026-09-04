package desktop

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/hostsettings"
)

// This file is an in-package test on purpose: the two host actions it covers
// are the only places where this application can make the operating system open
// something, and proving what they can and cannot be made to open needs the
// injected opener rather than a real window.

func newHostActionBridge(t *testing.T, directory string) (*Bridge, *[]string) {
	t.Helper()
	opened := []string{}
	bridge := NewBridgeWithDependencies(Dependencies{
		ApplicationVersion: "2.0.0",
		HostSettings:       hostsettings.NewStore(directory),
	})
	bridge.openHostURL = func(_ context.Context, url string) { opened = append(opened, url) }
	bridge.Startup(context.Background())
	return bridge, &opened
}

// TestOpenHostLocationAcceptsOnlyKnownIdentifiers is the whole point of the
// identifier-based design: the frontend can open the configured state
// directory, while the unavailable local-log destination and arbitrary values
// are rejected.
func TestOpenHostLocationAcceptsOnlyKnownIdentifiers(t *testing.T) {
	directory := t.TempDir()
	bridge, opened := newHostActionBridge(t, directory)

	if err := bridge.OpenHostLocation("configuration"); err != nil {
		t.Fatalf("OpenHostLocation(configuration): %v", err)
	}
	if len(*opened) != 1 || (*opened)[0] != "file://"+directory {
		t.Fatalf("opened = %v", *opened)
	}

	for _, refused := range []string{
		"",
		"logs",
		"unknown",
		"/etc/passwd",
		"file:///etc/passwd",
		"https://example.com",
		"../../../etc",
	} {
		if err := bridge.OpenHostLocation(refused); err == nil {
			t.Fatalf("OpenHostLocation(%q) was accepted", refused)
		}
	}
	if len(*opened) != 1 {
		t.Fatalf("a refused location still opened something: %v", *opened)
	}
}

// TestOpenProjectLinkResolvesOnlyApprovedIdentifiers states the same property
// for the outward-facing action: the frontend never supplies an address.
func TestOpenProjectLinkResolvesOnlyApprovedIdentifiers(t *testing.T) {
	bridge, opened := newHostActionBridge(t, t.TempDir())

	if err := bridge.OpenProjectLink("repository"); err != nil {
		t.Fatalf("OpenProjectLink(repository): %v", err)
	}
	if len(*opened) != 1 || (*opened)[0] != "https://github.com/oisis/EldenRing-SaveEditor" {
		t.Fatalf("opened = %v", *opened)
	}

	for _, refused := range []string{"", "unknown", "https://example.com", "file:///etc/passwd"} {
		if err := bridge.OpenProjectLink(refused); err == nil {
			t.Fatalf("OpenProjectLink(%q) was accepted", refused)
		}
	}
	if len(*opened) != 1 {
		t.Fatalf("a refused link still opened something: %v", *opened)
	}
}

// TestOpenHostLocationRefusesAHostWithoutADirectory keeps the action honest on
// a host that has no state directory: it reports that there is nothing to open
// instead of opening the process's working directory.
func TestOpenHostLocationRefusesAHostWithoutADirectory(t *testing.T) {
	opened := []string{}
	bridge := NewBridgeWithDependencies(Dependencies{
		ApplicationVersion: "2.0.0",
		HostSettings:       hostsettings.NewStore(""),
	})
	bridge.openHostURL = func(_ context.Context, url string) { opened = append(opened, url) }
	bridge.Startup(context.Background())

	if err := bridge.OpenHostLocation("configuration"); err == nil {
		t.Fatal("OpenHostLocation succeeded on a host with no state directory")
	}
	if len(opened) != 0 {
		t.Fatalf("opened = %v", opened)
	}
}

// TestCancelDeploymentOperationCancelsOnlyTheNamedOperation covers the
// cancellation registry: one operation is reachable by its identifier, and
// cancelling an unknown one is an ordinary outcome rather than a failure.
func TestCancelDeploymentOperationCancelsOnlyTheNamedOperation(t *testing.T) {
	bridge := NewBridgeWithDependencies(Dependencies{ApplicationVersion: "2.0.0"})
	bridge.Startup(context.Background())

	first, releaseFirst := bridge.operationContext("operation-1")
	second, releaseSecond := bridge.operationContext("operation-2")
	defer releaseFirst()
	defer releaseSecond()

	if err := bridge.CancelDeploymentOperation("operation-1"); err != nil {
		t.Fatalf("CancelDeploymentOperation: %v", err)
	}
	if first.Err() == nil {
		t.Fatal("the named operation was not cancelled")
	}
	if second.Err() != nil {
		t.Fatal("cancelling one operation cancelled another")
	}
	if err := bridge.CancelDeploymentOperation("operation-that-never-ran"); err != nil {
		t.Fatalf("cancelling an unknown operation reported %v", err)
	}
}

// TestReleaseDeploymentStagingAcceptsOnlyTrackedFiles protects the cleanup
// boundary: a downloaded staging directory is removed, while an arbitrary path
// supplied by the frontend is rejected and left untouched.
func TestReleaseDeploymentStagingAcceptsOnlyTrackedFiles(t *testing.T) {
	bridge := NewBridgeWithDependencies(Dependencies{ApplicationVersion: "2.0.0"})
	staging := t.TempDir()
	stagedPath := filepath.Join(staging, "downloaded.sl2")
	if err := os.WriteFile(stagedPath, []byte("staged"), 0o600); err != nil {
		t.Fatalf("write staged file: %v", err)
	}
	bridge.trackStagedDownload(stagedPath)
	if err := bridge.ReleaseDeploymentStaging(stagedPath); err != nil {
		t.Fatalf("ReleaseDeploymentStaging: %v", err)
	}
	if _, err := os.Stat(staging); !os.IsNotExist(err) {
		t.Fatalf("staging directory still exists or could not be checked: %v", err)
	}

	unknownDirectory := t.TempDir()
	unknownPath := filepath.Join(unknownDirectory, "keep.sl2")
	if err := os.WriteFile(unknownPath, []byte("keep"), 0o600); err != nil {
		t.Fatalf("write unknown file: %v", err)
	}
	if err := bridge.ReleaseDeploymentStaging(unknownPath); err == nil {
		t.Fatal("ReleaseDeploymentStaging accepted an untracked path")
	}
	if _, err := os.Stat(unknownPath); err != nil {
		t.Fatalf("an untracked path was changed: %v", err)
	}
}

// TestDesktopBuildTemplateSelectionMatchesTheSupportedExport proves the
// desktop action is not the empty selection the endpoint must reject and does
// not include the two unconfirmed spell slots.
func TestDesktopBuildTemplateSelectionMatchesTheSupportedExport(t *testing.T) {
	selection := desktopBuildTemplateSelection()
	if !selection.HasAnySelected() || selection.Profile == nil || selection.Stats == nil || selection.Spells == nil {
		t.Fatalf("selection = %+v", selection)
	}
	if len(selection.Profile.Fields) != 2 || len(selection.Stats.Fields) != 8 || len(selection.Spells.Fields) != 12 {
		t.Fatalf("field counts = %d/%d/%d, want 2/8/12",
			len(selection.Profile.Fields), len(selection.Stats.Fields), len(selection.Spells.Fields))
	}
	if selection.Spells.Fields["spell13"] || selection.Spells.Fields["spell14"] {
		t.Fatal("the desktop selection includes an unsupported spell slot")
	}
}
