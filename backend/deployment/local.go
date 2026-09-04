package deployment

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// localDriver implements Driver against the filesystem of the machine SaveForge
// runs on. A "local target" is another game installation on the same computer,
// so every operation is an ordinary file operation.
type localDriver struct{ target Target }

func localDir(path string) string { return filepath.Dir(path) }

func (driver *localDriver) Kind() Kind { return KindLocal }

// Test proves the target directory exists and can be written, without creating,
// replacing or removing anything the user owns.
func (driver *localDriver) Test(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	directory := filepath.Dir(driver.target.SavePath)
	info, err := os.Stat(directory)
	if err != nil {
		return fmt.Errorf("the target directory is not reachable: %w", err)
	}
	if !info.IsDir() {
		return errors.New("the target save path does not sit inside a directory")
	}
	probe, err := os.CreateTemp(directory, ".saveforge-probe-")
	if err != nil {
		return fmt.Errorf("the target directory is not writable: %w", err)
	}
	name := probe.Name()
	_ = probe.Close()
	return os.Remove(name)
}

func (driver *localDriver) Exists(ctx context.Context, path string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	info, err := os.Lstat(path)
	switch {
	case err == nil:
		if !info.Mode().IsRegular() {
			return false, fmt.Errorf("the target path is not a regular file")
		}
		return true, nil
	case errors.Is(err, os.ErrNotExist):
		return false, nil
	default:
		return false, fmt.Errorf("cannot inspect the target path: %w", err)
	}
}

func (driver *localDriver) CopyOnTarget(ctx context.Context, source, destination string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return copyFileExclusive(source, destination)
}

func (driver *localDriver) FilesEqual(ctx context.Context, left, right string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	leftDigest, err := fileDigest(left)
	if err != nil {
		return false, err
	}
	rightDigest, err := fileDigest(right)
	if err != nil {
		return false, err
	}
	return leftDigest == rightDigest, nil
}

func (driver *localDriver) Remove(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("cannot remove the target file: %w", err)
	}
	return nil
}

// ReplaceFromLocal writes the prepared file beside the target under a temporary
// name, verifies it byte for byte, and only then renames it over the save. The
// rename is the single irreversible point of the operation, exactly as section 6
// of deployment.md describes it, and every failure before it leaves the existing
// save untouched.
func (driver *localDriver) ReplaceFromLocal(ctx context.Context, localPath, targetPath string) (ReplacementResult, error) {
	return driver.replace(ctx, localPath, targetPath)
}

// ReplaceOnTarget is the same replacement with a source that is already on the
// target. For a local target the two are literally the same operation, so they
// share one implementation instead of two that could drift apart.
func (driver *localDriver) ReplaceOnTarget(ctx context.Context, sourcePath, targetPath string) (ReplacementResult, error) {
	return driver.replace(ctx, sourcePath, targetPath)
}

func (driver *localDriver) replace(ctx context.Context, localPath, targetPath string) (ReplacementResult, error) {
	if err := ctx.Err(); err != nil {
		return ReplacementResult{}, err
	}
	expected, err := fileDigest(localPath)
	if err != nil {
		return ReplacementResult{}, err
	}
	staged, err := copyFileToUniqueTemp(localPath, filepath.Dir(targetPath))
	if err != nil {
		return ReplacementResult{}, err
	}
	defer os.Remove(staged)
	staging, err := fileDigest(staged)
	if err != nil {
		return ReplacementResult{}, err
	}
	if staging != expected {
		return ReplacementResult{}, errors.New("the transferred file does not match the prepared save")
	}
	if err := ctx.Err(); err != nil {
		return ReplacementResult{}, err
	}
	if err := os.Rename(staged, targetPath); err != nil {
		return ReplacementResult{}, fmt.Errorf("cannot replace the target save: %w", err)
	}
	// Past the replacement point the operation always finishes its verification
	// and reports the real state rather than aborting half way.
	written, err := fileDigest(targetPath)
	if err != nil {
		return ReplacementResult{Committed: true}, err
	}
	if written != expected {
		return ReplacementResult{Committed: true}, errors.New("the replaced target save does not match the prepared save")
	}
	return ReplacementResult{Committed: true, Verified: true}, nil
}

func (driver *localDriver) CopyToLocal(ctx context.Context, targetPath, localPath string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return copyFile(targetPath, localPath)
}

func (driver *localDriver) RunStart(ctx context.Context) (CommandOutcome, error) {
	return driver.runCommand(ctx, driver.target.StartCommand, "start command")
}

func (driver *localDriver) RunStop(ctx context.Context) (CommandOutcome, error) {
	return driver.runCommand(ctx, driver.target.StopCommand, "stop command")
}

// runCommand executes exactly the command line the user configured.
//
// The command reaches the shell as one argument, never as a string this code
// assembled: no path, target name or file name is ever concatenated into it. A
// non-zero exit is reported as an outcome rather than raised as an error,
// because "the stop command found no process" is a legitimate result the caller
// has to be able to see.
func (driver *localDriver) runCommand(
	ctx context.Context, command string, field string,
) (CommandOutcome, error) {
	if command == "" {
		return CommandOutcome{Configured: false, Detail: "no " + field + " is configured"}, nil
	}
	if err := ctx.Err(); err != nil {
		return CommandOutcome{}, err
	}
	shell, flag := "/bin/sh", "-c"
	if runtime.GOOS == "windows" {
		shell, flag = "cmd", "/c"
	}
	execution := exec.CommandContext(ctx, shell, flag, command)
	err := execution.Run()
	outcome := CommandOutcome{Configured: true, Executed: true}
	if err == nil {
		return outcome, nil
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		outcome.ExitCode = exitError.ExitCode()
		outcome.Detail = fmt.Sprintf("the %s ended with exit code %d", field, outcome.ExitCode)
		return outcome, nil
	}
	// The command could not be started at all, which is a configuration fault
	// rather than a result of the game's state.
	return CommandOutcome{Configured: true}, fmt.Errorf("the %s could not be run", field)
}

// GameStatus reports what the backend can actually confirm.
//
// Identifying the game process needs a contract that does not exist: the target
// configuration carries a start and a stop command and nothing that names a
// process, and deployment.md defines none. Guessing from a process list or from
// the save's modification time would be a heuristic inventing a state, so the
// driver states the truth — unknown — and the interface applies the explicit
// warning and confirmation that section 4 defines for exactly this case.
func (driver *localDriver) GameStatus(context.Context) (GameStatus, error) {
	return GameUnknown, nil
}

// WaitForStableSave waits until the size and modification time of the target
// save are unchanged across several consecutive observations. A missing file is
// stable by definition.
func (driver *localDriver) WaitForStableSave(ctx context.Context, path string) error {
	previous := ""
	stable := 0
	for stable < stabilisationRounds {
		if err := ctx.Err(); err != nil {
			return err
		}
		current := ""
		info, err := os.Stat(path)
		switch {
		case err == nil:
			current = fmt.Sprintf("%d:%d", info.Size(), info.ModTime().UnixNano())
		case errors.Is(err, os.ErrNotExist):
			current = "absent"
		default:
			return fmt.Errorf("cannot observe the target save: %w", err)
		}
		if current == previous {
			stable++
		} else {
			stable = 1
			previous = current
		}
		if stable < stabilisationRounds {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(stabilisationPoll):
			}
		}
	}
	return nil
}

func (driver *localDriver) Close() error { return nil }

// copyFile copies source onto destination through a fresh file, flushing it
// before it is considered written.
func copyFile(source string, destination string) error {
	return copyFileWithFlags(source, destination, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
}

func copyFileExclusive(source string, destination string) error {
	return copyFileWithFlags(source, destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL)
}

func copyFileWithFlags(source string, destination string, flags int) error {
	reader, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("cannot read the source file: %w", err)
	}
	defer reader.Close()
	info, err := reader.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("the source is not a regular file")
	}
	writer, err := os.OpenFile(destination, flags, info.Mode().Perm())
	if err != nil {
		return fmt.Errorf("cannot create the destination file: %w", err)
	}
	if _, err := io.Copy(writer, reader); err != nil {
		_ = writer.Close()
		_ = os.Remove(destination)
		return fmt.Errorf("cannot copy the file: %w", err)
	}
	if err := writer.Sync(); err != nil {
		_ = writer.Close()
		_ = os.Remove(destination)
		return err
	}
	return writer.Close()
}

func copyFileToUniqueTemp(source, directory string) (string, error) {
	reader, err := os.Open(source)
	if err != nil {
		return "", fmt.Errorf("cannot read the source file: %w", err)
	}
	defer reader.Close()
	info, err := reader.Stat()
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("the source is not a regular file")
	}
	writer, err := os.CreateTemp(directory, ".saveforge-incoming-*")
	if err != nil {
		return "", fmt.Errorf("cannot create the target staging file: %w", err)
	}
	name := writer.Name()
	remove := true
	defer func() {
		_ = writer.Close()
		if remove {
			_ = os.Remove(name)
		}
	}()
	if err := writer.Chmod(info.Mode().Perm()); err != nil {
		return "", err
	}
	if _, err := io.Copy(writer, reader); err != nil {
		return "", fmt.Errorf("cannot copy the target staging file: %w", err)
	}
	if err := writer.Sync(); err != nil {
		return "", fmt.Errorf("cannot flush the target staging file: %w", err)
	}
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("cannot close the target staging file: %w", err)
	}
	remove = false
	return name, nil
}

func fileDigest(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("cannot verify the file: %w", err)
	}
	defer file.Close()
	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return "", fmt.Errorf("cannot verify the file: %w", err)
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}
