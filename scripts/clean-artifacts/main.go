package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const expectedModule = "module github.com/oisis/EldenRing-SaveForge"

var artifactPaths = []string{
	"app_version_generated.go",
	"build/bin",
	"frontend/dist",
	".cache",
	".parcel-cache",
	".npm",
	".eslintcache",
	".stylelintcache",
	"frontend/.cache",
	"frontend/.vite",
	"frontend/coverage",
	"frontend/.eslintcache",
	"frontend/.stylelintcache",
	"frontend/tsconfig.tsbuildinfo",
	"frontend/node_modules/.vite",
	"Elden Ring SaveForge",
	"Elden Ring SaveForge.exe",
	"EldenRing-SaveForge",
	"EldenRing-SaveForge.exe",
	"EldenRing-SaveEditor",
	"EldenRing-SaveEditor.exe",
	"main",
	"main.exe",
}

func main() {
	dryRun := flag.Bool("dry-run", false, "print artifact paths without removing them")
	flag.Parse()

	root, err := os.Getwd()
	if err != nil {
		exitWithError("resolve repository root: %v", err)
	}
	root, err = filepath.Abs(root)
	if err != nil {
		exitWithError("resolve absolute repository root: %v", err)
	}
	if err := validateRepositoryRoot(root); err != nil {
		exitWithError("%v", err)
	}
	if err := cleanArtifacts(root, artifactPaths, *dryRun, os.Stdout); err != nil {
		exitWithError("%v", err)
	}
}

func validateRepositoryRoot(root string) error {
	content, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return fmt.Errorf("verify repository root: %w", err)
	}
	firstLine := strings.SplitN(string(content), "\n", 2)[0]
	if strings.TrimSpace(firstLine) != expectedModule {
		return fmt.Errorf("unexpected Go module %q", strings.TrimSpace(firstLine))
	}
	return nil
}

func cleanArtifacts(root string, relativePaths []string, dryRun bool, output io.Writer) error {
	targets := make([]string, len(relativePaths))
	for index, relativePath := range relativePaths {
		target, err := artifactTarget(root, relativePath)
		if err != nil {
			return err
		}
		targets[index] = target
	}

	for index, relativePath := range relativePaths {
		if dryRun {
			fmt.Fprintln(output, relativePath)
			continue
		}
		if err := os.RemoveAll(targets[index]); err != nil {
			return fmt.Errorf("remove %q: %w", relativePath, err)
		}
	}
	return nil
}

func artifactTarget(root, relativePath string) (string, error) {
	cleanPath := filepath.Clean(relativePath)
	if cleanPath == "." || filepath.IsAbs(cleanPath) ||
		cleanPath == ".." || strings.HasPrefix(cleanPath, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("unsafe artifact path %q", relativePath)
	}
	if cleanPath == "tmp" || strings.HasPrefix(cleanPath, "tmp"+string(filepath.Separator)) {
		return "", fmt.Errorf("refusing to clean protected path %q", relativePath)
	}

	target := filepath.Join(root, cleanPath)
	resolvedPath, err := filepath.Rel(root, target)
	if err != nil {
		return "", fmt.Errorf("validate artifact path %q: %w", relativePath, err)
	}
	if resolvedPath == "." || resolvedPath == ".." ||
		strings.HasPrefix(resolvedPath, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("artifact path escapes repository root: %q", relativePath)
	}
	return target, nil
}

func exitWithError(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "clean artifacts: "+format+"\n", args...)
	os.Exit(1)
}
