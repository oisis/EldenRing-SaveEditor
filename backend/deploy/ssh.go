package deploy

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// SSHManager handles SSH connections and remote operations for deploy targets.
type SSHManager struct {
	store *TargetStore
}

// NewSSHManager creates a new SSH manager backed by the given target store.
func NewSSHManager(store *TargetStore) *SSHManager {
	return &SSHManager{store: store}
}

// TestConnection verifies SSH connectivity to a target. Returns host info on success.
func (m *SSHManager) TestConnection(targetName string) (string, error) {
	t, ok := m.store.Get(targetName)
	if !ok {
		return "", fmt.Errorf("target %q not found", targetName)
	}
	client, err := m.dial(t)
	if err != nil {
		return "", err
	}
	defer client.Close()
	return fmt.Sprintf("Connected to %s@%s:%d", t.User, t.Host, t.Port), nil
}

// UploadSave uploads a local save file to the remote target.
// It creates a timestamped backup of the remote file before overwriting.
func (m *SSHManager) UploadSave(targetName string, localPath string) error {
	t, ok := m.store.Get(targetName)
	if !ok {
		return fmt.Errorf("target %q not found", targetName)
	}

	localData, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("cannot read local file: %w", err)
	}

	client, err := m.dial(t)
	if err != nil {
		return err
	}
	defer client.Close()

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return fmt.Errorf("SFTP session failed: %w", err)
	}
	defer sftpClient.Close()

	// Backup remote file if it exists
	if _, statErr := sftpClient.Stat(t.SavePath); statErr == nil {
		backupPath := fmt.Sprintf("%s.%s.bkp", t.SavePath, time.Now().Format("20060102_150405"))
		src, err := sftpClient.Open(t.SavePath)
		if err == nil {
			dst, err := sftpClient.Create(backupPath)
			if err == nil {
				io.Copy(dst, src)
				dst.Close()
			}
			src.Close()
		}
	}

	// Ensure remote directory exists (use path.Dir for POSIX remote paths)
	remoteDir := path.Dir(t.SavePath)
	sftpClient.MkdirAll(remoteDir)

	// Upload
	dst, err := sftpClient.Create(t.SavePath)
	if err != nil {
		return fmt.Errorf("cannot create remote file %s: %w", t.SavePath, err)
	}
	defer dst.Close()

	n, err := dst.Write(localData)
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}

	// Verify size
	if n != len(localData) {
		return fmt.Errorf("size mismatch: wrote %d, expected %d", n, len(localData))
	}

	return nil
}

// DownloadSave downloads the save file from the remote target to a local path.
func (m *SSHManager) DownloadSave(targetName string, localPath string) error {
	t, ok := m.store.Get(targetName)
	if !ok {
		return fmt.Errorf("target %q not found", targetName)
	}

	client, err := m.dial(t)
	if err != nil {
		return err
	}
	defer client.Close()

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return fmt.Errorf("SFTP session failed: %w", err)
	}
	defer sftpClient.Close()

	src, err := sftpClient.Open(t.SavePath)
	if err != nil {
		return fmt.Errorf("cannot open remote file: %w", err)
	}
	defer src.Close()

	// Ensure local directory exists
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return fmt.Errorf("cannot create local directory: %w", err)
	}

	dst, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("cannot create local file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	return nil
}

// LaunchGame executes the game start command on the remote target.
func (m *SSHManager) LaunchGame(targetName string) (string, error) {
	t, ok := m.store.Get(targetName)
	if !ok {
		return "", fmt.Errorf("target %q not found", targetName)
	}
	cmd := t.GameStartCmd
	if cmd == "" {
		cmd = DefaultStartCmd
	}
	return m.execRemote(t, cmd)
}

// CloseGame executes the game stop command on the remote target.
func (m *SSHManager) CloseGame(targetName string) (string, error) {
	t, ok := m.store.Get(targetName)
	if !ok {
		return "", fmt.Errorf("target %q not found", targetName)
	}
	cmd := t.GameStopCmd
	if cmd == "" {
		cmd = DefaultStopCmd
	}
	return m.execRemote(t, cmd)
}

// DeployAndLaunch performs the full workflow: close game → wait → upload → launch.
func (m *SSHManager) DeployAndLaunch(targetName string, localPath string) error {
	// Step 1: Close game (ignore errors — game might not be running)
	m.CloseGame(targetName)

	// Step 2: Wait for graceful shutdown
	time.Sleep(3 * time.Second)

	// Step 3: Upload save
	if err := m.UploadSave(targetName, localPath); err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}

	// Step 4: Launch game
	if _, err := m.LaunchGame(targetName); err != nil {
		return fmt.Errorf("launch failed: %w", err)
	}

	return nil
}

func (m *SSHManager) dial(t Target) (*ssh.Client, error) {
	keyPath := expandHome(t.KeyPath)

	var authMethods []ssh.AuthMethod

	if keyPath != "" {
		keyData, err := os.ReadFile(keyPath)
		if err != nil {
			return nil, fmt.Errorf("cannot read SSH key %s: %w", keyPath, err)
		}
		signer, err := ssh.ParsePrivateKey(keyData)
		if err != nil {
			return nil, fmt.Errorf("cannot parse SSH key: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	// Fallback: try SSH agent
	if len(authMethods) == 0 {
		return nil, fmt.Errorf("no SSH key configured for target %q", t.Name)
	}

	config := &ssh.ClientConfig{
		User:            t.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", t.Host, t.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("SSH connection to %s failed: %w", addr, err)
	}
	return client, nil
}

func (m *SSHManager) execRemote(t Target, command string) (string, error) {
	client, err := m.dial(t)
	if err != nil {
		return "", err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("SSH session failed: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)
	if err != nil {
		return string(output), fmt.Errorf("command failed: %w — output: %s", err, string(output))
	}
	return strings.TrimSpace(string(output)), nil
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}
