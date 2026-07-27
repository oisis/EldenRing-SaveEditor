package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestArtifactTarget(t *testing.T) {
	root := t.TempDir()

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{name: "build output", path: "build/bin"},
		{name: "frontend cache", path: "frontend/.vite"},
		{name: "root", path: ".", wantErr: true},
		{name: "absolute", path: filepath.Join(root, "outside"), wantErr: true},
		{name: "parent", path: "../outside", wantErr: true},
		{name: "tmp root", path: "tmp", wantErr: true},
		{name: "tmp child", path: "tmp/cache", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := artifactTarget(root, test.path)
			if test.wantErr && err == nil {
				t.Fatalf("artifactTarget(%q) succeeded, want error", test.path)
			}
			if !test.wantErr && err != nil {
				t.Fatalf("artifactTarget(%q) returned error: %v", test.path, err)
			}
		})
	}
}

func TestCleanArtifactsRemovesListedPathsAndLeavesTmpUntouched(t *testing.T) {
	root := t.TempDir()
	buildArtifact := filepath.Join(root, "build", "bin", "app")
	protectedArtifact := filepath.Join(root, "tmp", "keep.txt")
	for _, artifactFile := range []string{buildArtifact, protectedArtifact} {
		if err := os.MkdirAll(filepath.Dir(artifactFile), 0o755); err != nil {
			t.Fatalf("create fixture directory: %v", err)
		}
		if err := os.WriteFile(artifactFile, []byte("fixture"), 0o600); err != nil {
			t.Fatalf("create fixture: %v", err)
		}
	}

	if err := cleanArtifacts(root, []string{"build/bin"}, false, &bytes.Buffer{}); err != nil {
		t.Fatalf("cleanArtifacts returned error: %v", err)
	}
	if _, err := os.Stat(buildArtifact); !os.IsNotExist(err) {
		t.Fatalf("build artifact still exists or stat failed unexpectedly: %v", err)
	}
	if _, err := os.Stat(protectedArtifact); err != nil {
		t.Fatalf("tmp artifact was modified: %v", err)
	}
}

func TestCleanArtifactsValidatesAllPathsBeforeRemovingAnything(t *testing.T) {
	root := t.TempDir()
	buildArtifact := filepath.Join(root, "build", "bin", "app")
	if err := os.MkdirAll(filepath.Dir(buildArtifact), 0o755); err != nil {
		t.Fatalf("create fixture directory: %v", err)
	}
	if err := os.WriteFile(buildArtifact, []byte("fixture"), 0o600); err != nil {
		t.Fatalf("create fixture: %v", err)
	}

	err := cleanArtifacts(root, []string{"build/bin", "tmp/cache"}, false, &bytes.Buffer{})
	if err == nil {
		t.Fatal("cleanArtifacts succeeded, want protected-path error")
	}
	if _, err := os.Stat(buildArtifact); err != nil {
		t.Fatalf("valid artifact was removed before validation completed: %v", err)
	}
}

func TestCleanArtifactsDryRunPropagatesWriterError(t *testing.T) {
	err := cleanArtifacts(t.TempDir(), []string{"build/bin"}, true, failingWriter{})
	if err == nil {
		t.Fatal("cleanArtifacts succeeded, want writer error")
	}
}
