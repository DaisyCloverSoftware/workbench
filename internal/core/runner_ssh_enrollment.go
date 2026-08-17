package core

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const runnerSSHDefaultIdentityName = "id_ed25519"

// PrepareRunnerSSHEnrollment creates (or reuses) the desktop user's normal
// OpenSSH Ed25519 identity and returns one ready-to-send operator prompt that
// authorises only the public half on the existing runner account. Workbench
// never exposes the private key through task state, MCP, relay, logs or prompts.
//
// We deliberately use OpenSSH's standard id_ed25519 location here. Both the
// interactive console and the existing unattended BatchMode transport already
// use OpenSSH's normal identity discovery, so once the public key is authorised
// no additional secret path or command-line setting is required.
func PrepareRunnerSSHEnrollment(host string) (string, error) {
	validated, err := validateSSHHostTarget(host)
	if err != nil {
		return "", err
	}
	publicKey, err := ensureRunnerSSHDefaultIdentity()
	if err != nil {
		return "", err
	}
	account := validated
	if at := strings.IndexByte(validated, '@'); at >= 0 {
		account = validated[:at]
	}
	return runnerSSHEnrollmentPrompt(account, publicKey), nil
}

func ensureRunnerSSHDefaultIdentity() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locate Windows user profile for SSH key: %w", err)
	}
	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		return "", fmt.Errorf("create SSH key directory: %w", err)
	}
	privatePath := filepath.Join(sshDir, runnerSSHDefaultIdentityName)
	publicPath := privatePath + ".pub"

	if info, statErr := os.Stat(privatePath); statErr != nil {
		if !errors.Is(statErr, os.ErrNotExist) {
			return "", fmt.Errorf("inspect SSH private key: %w", statErr)
		}
		cmd := exec.Command("ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-f", privatePath, "-C", "workbench-runner")
		configureChildProcess(cmd, false)
		if out, runErr := cmd.CombinedOutput(); runErr != nil {
			message := strings.TrimSpace(string(out))
			if message == "" {
				message = runErr.Error()
			}
			return "", fmt.Errorf("create unattended Workbench SSH key with ssh-keygen: %s", message)
		}
	} else if info.IsDir() {
		return "", errors.New("SSH identity path is a directory, not a private key")
	}

	publicBytes, readErr := os.ReadFile(publicPath)
	if readErr != nil {
		// A private key can legitimately outlive its .pub companion. Re-derive
		// the public half locally instead of replacing or exposing the private key.
		cmd := exec.Command("ssh-keygen", "-y", "-f", privatePath)
		configureChildProcess(cmd, false)
		out, runErr := cmd.CombinedOutput()
		if runErr != nil {
			message := strings.TrimSpace(string(out))
			if message == "" {
				message = runErr.Error()
			}
			return "", fmt.Errorf("read existing SSH public key: %s", message)
		}
		publicBytes = append([]byte(strings.TrimSpace(string(out))+" workbench-runner"), '\n')
		if writeErr := os.WriteFile(publicPath, publicBytes, 0o644); writeErr != nil {
			return "", fmt.Errorf("restore SSH public-key file: %w", writeErr)
		}
	}
	return normalizeRunnerSSHPublicKey(string(publicBytes))
}

func normalizeRunnerSSHPublicKey(value string) (string, error) {
	if len(value) > 16<<10 {
		return "", errors.New("SSH public key is unexpectedly large")
	}
	fields := strings.Fields(strings.TrimSpace(value))
	if len(fields) < 2 {
		return "", errors.New("SSH public key is invalid")
	}
	if fields[0] != "ssh-ed25519" {
		return "", fmt.Errorf("Workbench runner bootstrap requires an Ed25519 SSH key, found %s", fields[0])
	}
	decoded, err := base64.StdEncoding.DecodeString(fields[1])
	if err != nil || len(decoded) < 32 || len(decoded) > 1024 {
		return "", errors.New("SSH public key payload is invalid")
	}
	return fields[0] + " " + fields[1] + " workbench-runner", nil
}

func runnerSSHEnrollmentPrompt(account, publicKey string) string {
	account = strings.TrimSpace(account)
	return "Authorize this Workbench desktop SSH public key for the existing runner account " + account + ".\n\n" +
		"This is the one-time bootstrap for unattended Workbench runner access. Preserve all existing SSH hardening: do NOT enable password authentication, root login, or weaken sshd, and do not replace existing authorized_keys entries.\n\n" +
		"For that account, ensure ~/.ssh exists with mode 700 and ~/.ssh/authorized_keys with mode 600, both owned by the account. Add the exact public key below once if it is not already present. Prefix the authorized_keys entry with: no-agent-forwarding,no-port-forwarding,no-X11-forwarding,no-user-rc\n\n" +
		"Then verify the key is present exactly once and report completion. Do not change Workbench/application code or any unrelated cluster configuration.\n\n" +
		"Public key:\n" + publicKey
}
