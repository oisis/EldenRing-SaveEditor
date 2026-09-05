// Package deployment owns SaveForge 2.0 deployment targets, the trusted SSH
// host keys, the target-side backup library and the operations that move a save
// between the application and a target.
//
// It is the single backend source of truth for all four. Save Manager is a
// second presentation of the same targets and the same backups, not a second
// model of them, and the frontend never performs a file or network operation of
// its own.
package deployment

import (
	"errors"
	"fmt"
	"net"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

// Kind states what a target is. The vocabulary is closed: deployment.md
// supports SSH targets and local filesystem targets and nothing else.
type Kind string

const (
	// KindLocal is a directory on the machine SaveForge itself runs on.
	KindLocal Kind = "local"
	// KindSSH is a remote machine reached over SSH with key authentication.
	KindSSH Kind = "ssh"
)

// Kinds is the closed vocabulary, in the order the interface offers it.
func Kinds() []Kind { return []Kind{KindLocal, KindSSH} }

// ParseKind validates one stated kind.
func ParseKind(value string) (Kind, error) {
	switch Kind(value) {
	case KindLocal:
		return KindLocal, nil
	case KindSSH:
		return KindSSH, nil
	}
	return "", fmt.Errorf("unknown deployment target kind %q; expected %q or %q",
		value, KindLocal, KindSSH)
}

// GameStatus is the backend's statement about the game on a target.
type GameStatus string

const (
	// GameRunning means the backend confirmed a running game.
	GameRunning GameStatus = "running"
	// GameStopped means the backend confirmed the game is not running.
	GameStopped GameStatus = "stopped"
	// GameUnknown means the backend could not establish either. Section 4 of
	// deployment.md defines this as a first-class state, not a failure: the
	// interface warns and the user may continue explicitly.
	GameUnknown GameStatus = "unknown"
)

// defaultSSHPort is the port a target that states none is reached on.
const defaultSSHPort = 22

// Target is one configured deployment destination.
//
// The SSH key is referenced by path only. Its contents are never read into this
// struct, never copied into the configuration and never written to a log or a
// diagnostic report, which is what section 2 of deployment.md requires.
type Target struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Kind Kind   `json:"kind"`
	// SavePath is the absolute path of the save file on the target system.
	SavePath string `json:"savePath"`
	// StartCommand, StopCommand and StatusCommand are the user's own explicitly
	// configured commands. The backend runs them as configured and never
	// assembles one by concatenating a path or a name into a shell string.
	StartCommand string `json:"startCommand,omitempty"`
	StopCommand  string `json:"stopCommand,omitempty"`
	// StatusCommand is the only way this application learns whether the game is
	// running on a target. Its contract is the exit code and nothing else: 0
	// means running, 1 means stopped, and every other outcome — no command,
	// another exit code, a timeout or a transport fault — is unknown. A target
	// configured before this field existed simply reports unknown.
	StatusCommand string `json:"statusCommand,omitempty"`

	// Host, Port, User and KeyPath apply to an SSH target only.
	Host    string `json:"host,omitempty"`
	Port    int    `json:"port,omitempty"`
	User    string `json:"user,omitempty"`
	KeyPath string `json:"keyPath,omitempty"`
}

// Address is the "host:port" the trusted host key is remembered under and the
// address the transport dials. It is built with net.JoinHostPort so an IPv6
// literal is bracketed: a fingerprint must be remembered under exactly the
// address the connection used, and "::1:22" is not that address.
func (target Target) Address() string {
	port := target.Port
	if port == 0 {
		port = defaultSSHPort
	}
	return net.JoinHostPort(target.Host, strconv.Itoa(port))
}

// BackupDirectory is the directory on the target the backups of its save live
// in. It is derived from the save path rather than configured separately, so a
// target has exactly one backup location and it always sits beside the save it
// protects.
func (target Target) BackupDirectory() string {
	if target.Kind == KindLocal {
		return localDir(target.SavePath)
	}
	return path.Dir(target.SavePath)
}

// Validate rejects an incomplete or unsafe target before it is stored.
//
// The rules fail closed: an SSH target without a key path is refused rather
// than quietly falling back to an agent, a password prompt or a default key,
// because deployment.md allows key authentication only.
func (target Target) Validate() error {
	if strings.TrimSpace(target.Name) == "" {
		return errors.New("a deployment target needs a name")
	}
	if _, err := ParseKind(string(target.Kind)); err != nil {
		return err
	}
	if strings.TrimSpace(target.SavePath) == "" {
		return errors.New("a deployment target needs the save path on the target system")
	}
	if strings.ContainsAny(target.SavePath, "\x00\n\r") {
		return errors.New("the save path contains characters a path cannot carry")
	}
	if target.Kind == KindLocal && !filepath.IsAbs(target.SavePath) {
		return errors.New("a local deployment save path must be absolute")
	}
	if target.Kind == KindSSH && !path.IsAbs(target.SavePath) {
		return errors.New("an SSH deployment save path must be an absolute POSIX path")
	}
	if err := validateCommand("start command", target.StartCommand); err != nil {
		return err
	}
	if err := validateCommand("stop command", target.StopCommand); err != nil {
		return err
	}
	if err := validateCommand("status command", target.StatusCommand); err != nil {
		return err
	}
	if target.Kind != KindSSH {
		return nil
	}
	if strings.TrimSpace(target.Host) == "" {
		return errors.New("an SSH target needs a host")
	}
	if target.Port < 0 || target.Port > 65535 {
		return fmt.Errorf("an SSH port must be between 0 and 65535; got %d", target.Port)
	}
	if strings.TrimSpace(target.User) == "" {
		return errors.New("an SSH target needs a user")
	}
	if strings.TrimSpace(target.KeyPath) == "" {
		return errors.New("an SSH target needs the path of its private key; passwords are not supported")
	}
	return nil
}

// validateCommand rejects a command line the backend could not run as one
// deliberate command. A newline or a NUL byte would let one configured command
// become several, so it is refused at configuration time rather than at run
// time.
func validateCommand(field string, command string) error {
	if command == "" {
		return nil
	}
	if strings.ContainsAny(command, "\x00\n\r") {
		return fmt.Errorf("the %s must be a single command line", field)
	}
	return nil
}
