package application

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestMain(m *testing.M) {
	originalDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "application tests: resolve working directory: %v\n", err)
		os.Exit(1)
	}

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		fmt.Fprintln(os.Stderr, "application tests: resolve test source path")
		os.Exit(1)
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
	if err := os.Chdir(repoRoot); err != nil {
		fmt.Fprintf(os.Stderr, "application tests: enter repository root: %v\n", err)
		os.Exit(1)
	}

	exitCode := m.Run()
	if err := os.Chdir(originalDir); err != nil {
		fmt.Fprintf(os.Stderr, "application tests: restore working directory: %v\n", err)
		if exitCode == 0 {
			exitCode = 1
		}
	}
	os.Exit(exitCode)
}

func TestWorkingDirectoryIsRepositoryRoot(t *testing.T) {
	if _, err := os.Stat("go.mod"); err != nil {
		t.Fatalf("application test working directory is not repository root: %v", err)
	}
}
