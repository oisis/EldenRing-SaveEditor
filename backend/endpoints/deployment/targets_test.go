package deployment

import (
	"testing"

	"github.com/oisis/EldenRing-SaveForge/backend/deployment"
)

// TestTargetEndpointsReportCapabilitiesAndTrustState covers the whole target
// surface in one flow: creating, listing, updating and deleting, plus the two
// facts the interface enables its actions from — whether this build can move a
// save to that kind of target, and whether an SSH host key was approved.
//
// No target is contacted: every call below reads or writes configuration only.
func TestTargetEndpointsReportCapabilitiesAndTrustState(t *testing.T) {
	store := deployment.NewStore(t.TempDir())

	empty, err := GetDeploymentTargets(store)
	if err != nil {
		t.Fatalf("GetDeploymentTargets: %v", err)
	}
	if len(empty.Targets) != 0 {
		t.Fatalf("targets = %+v, want none", empty.Targets)
	}
	if len(empty.AvailableKinds) != 2 {
		t.Fatalf("availableKinds = %v, want exactly local and ssh", empty.AvailableKinds)
	}

	withLocal, err := CreateDeploymentTarget(store, TargetInput{
		Name:     "Local copy",
		Kind:     "local",
		SavePath: "/saves/ER0000.sl2",
	})
	if err != nil {
		t.Fatalf("CreateDeploymentTarget: %v", err)
	}
	if len(withLocal.Targets) != 1 || !withLocal.Targets[0].TransferSupported {
		t.Fatalf("targets = %+v, want a local target this build can use", withLocal.Targets)
	}

	both, err := CreateDeploymentTarget(store, TargetInput{
		Name:     "Steam Deck",
		Kind:     "ssh",
		SavePath: "/home/deck/ER0000.sl2",
		Host:     "192.0.2.1",
		Port:     22,
		User:     "deck",
		KeyPath:  "/home/user/.ssh/id_ed25519",
	})
	if err != nil {
		t.Fatalf("CreateDeploymentTarget: %v", err)
	}
	var ssh TargetEntry
	for _, entry := range both.Targets {
		if entry.Kind == deployment.KindSSH {
			ssh = entry
		}
	}
	if ssh.ID == "" {
		t.Fatalf("targets = %+v, want the SSH target", both.Targets)
	}
	// The interface disables the operations from this flag instead of assuming
	// what a target kind can do.
	if ssh.TransferSupported || ssh.UnsupportedReason == "" {
		t.Fatalf("ssh entry = %+v, want it reported as unsupported with a reason", ssh)
	}
	if ssh.HostKeyTrusted {
		t.Fatal("a new SSH target already reports an approved host key")
	}

	trusted, err := TrustDeploymentHostKey(store, ssh.ID, "SHA256:abc")
	if err != nil {
		t.Fatalf("TrustDeploymentHostKey: %v", err)
	}
	for _, entry := range trusted.Targets {
		if entry.ID == ssh.ID && (!entry.HostKeyTrusted || entry.HostKeyFingerprint != "SHA256:abc") {
			t.Fatalf("ssh entry after trusting = %+v", entry)
		}
	}
	// A local target has no host key at all, so it can never be given one.
	local := withLocal.Targets[0]
	if _, err := TrustDeploymentHostKey(store, local.ID, "SHA256:abc"); err == nil {
		t.Fatal("TrustDeploymentHostKey accepted a local target")
	}

	// An incomplete SSH target is refused rather than stored with a hole in it.
	if _, err := CreateDeploymentTarget(store, TargetInput{
		Name: "No key", Kind: "ssh", SavePath: "/s.sl2", Host: "h", User: "u",
	}); err == nil {
		t.Fatal("CreateDeploymentTarget accepted an SSH target with no key path")
	}
	if _, err := CreateDeploymentTarget(store, TargetInput{
		Name: "Wrong kind", Kind: "ftp", SavePath: "/s.sl2",
	}); err == nil {
		t.Fatal("CreateDeploymentTarget accepted an unknown target kind")
	}

	renamed, err := UpdateDeploymentTarget(store, TargetInput{
		TargetID: local.ID,
		Name:     "Renamed",
		Kind:     "local",
		SavePath: "/saves/ER0000.sl2",
	})
	if err != nil {
		t.Fatalf("UpdateDeploymentTarget: %v", err)
	}
	found := false
	for _, entry := range renamed.Targets {
		if entry.ID == local.ID && entry.Name == "Renamed" {
			found = true
		}
	}
	if !found {
		t.Fatalf("targets after the update = %+v", renamed.Targets)
	}

	remaining, err := DeleteDeploymentTarget(store, local.ID)
	if err != nil {
		t.Fatalf("DeleteDeploymentTarget: %v", err)
	}
	if len(remaining.Targets) != 1 || remaining.Targets[0].ID != ssh.ID {
		t.Fatalf("targets after the delete = %+v", remaining.Targets)
	}
}

// TestGetTargetBackupsReportsTheCapabilityOfItsTarget: Save Manager enables its
// actions from the same backend statement Deployment does.
func TestGetTargetBackupsReportsTheCapabilityOfItsTarget(t *testing.T) {
	store := deployment.NewStore(t.TempDir())
	target, err := store.CreateTarget(deployment.Target{
		Name: "Steam Deck", Kind: deployment.KindSSH, SavePath: "/s.sl2",
		Host: "192.0.2.1", Port: 22, User: "deck", KeyPath: "/k",
	})
	if err != nil {
		t.Fatalf("CreateTarget: %v", err)
	}

	result, err := GetTargetBackups(store, target.ID)
	if err != nil {
		t.Fatalf("GetTargetBackups: %v", err)
	}
	if result.TransferSupported || result.UnsupportedReason == "" {
		t.Fatalf("result = %+v, want the unsupported statement", result)
	}
	if result.Backups == nil || len(result.Backups) != 0 {
		t.Fatalf("backups = %+v, want an empty list rather than a null", result.Backups)
	}
	if _, err := GetTargetBackups(store, "target-that-does-not-exist"); err == nil {
		t.Fatal("GetTargetBackups accepted an unknown target")
	}
}
