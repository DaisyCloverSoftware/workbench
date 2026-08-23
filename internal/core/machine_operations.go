package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	defaultMachineCommandTimeout = 90 * time.Second
	maxMachineCommandTimeout     = 10 * time.Minute
	maxMachineCommandOutputBytes = 1 << 20
	maxMachineCommandArgs        = 128
	maxMachineCommandArgBytes    = 2048
)

type MachineCommandRequest struct {
	Program        string   `json:"program"`
	Args           []string `json:"args,omitempty"`
	TimeoutSeconds int      `json:"timeout_seconds,omitempty"`
}

type MachineCommandResult struct {
	Program   string   `json:"program"`
	Args      []string `json:"args,omitempty"`
	Output    string   `json:"output,omitempty"`
	ReadOnly  bool     `json:"read_only"`
	ExitCode  int      `json:"exit_code"`
	Truncated bool     `json:"truncated,omitempty"`
	Transport string   `json:"transport,omitempty"`
}

// InspectMachine executes one explicitly allowlisted read-only machine command
// without a shell or an AI worker. It is the normal ChatGPT path for cluster and
// host diagnostics when Workbench itself is already running on the operator
// host.
func InspectMachine(ctx context.Context, req MachineCommandRequest) (MachineCommandResult, error) {
	readOnly, err := validateMachineCommand(req)
	if err != nil {
		return MachineCommandResult{}, err
	}
	if !readOnly {
		return MachineCommandResult{}, errors.New("machine command is mutating; use the explicit mutating machine-command tool")
	}
	return executeMachineCommand(ctx, req, true)
}

// RunMachineCommand executes one explicitly allowlisted mutating machine
// command. It deliberately accepts a program plus argv rather than shell text:
// ChatGPT can reason between calls, but Workbench never evaluates pipes,
// substitutions, redirections, command chains or arbitrary scripts here.
func RunMachineCommand(ctx context.Context, req MachineCommandRequest) (MachineCommandResult, error) {
	readOnly, err := validateMachineCommand(req)
	if err != nil {
		return MachineCommandResult{}, err
	}
	if readOnly {
		return MachineCommandResult{}, errors.New("machine command is read-only; use inspect_machine")
	}
	if err := validateOwnerGatedProductMutation(req); err != nil {
		return MachineCommandResult{}, err
	}
	return executeMachineCommand(ctx, req, false)
}

func validateMachineCommand(req MachineCommandRequest) (bool, error) {
	program := strings.ToLower(strings.TrimSpace(req.Program))
	if program == "" || program != strings.TrimSpace(req.Program) || strings.ContainsAny(program, `/\\`) {
		return false, errors.New("machine command program must be one allowlisted executable basename")
	}
	if len(req.Args) > maxMachineCommandArgs {
		return false, errors.New("machine command has too many arguments")
	}
	for _, arg := range req.Args {
		if len(arg) > maxMachineCommandArgBytes || strings.ContainsAny(arg, "\x00\r\n") {
			return false, errors.New("machine command argument is invalid or too large")
		}
	}
	joined := strings.Join(req.Args, " ")
	if LooksSecret(joined) {
		return false, errors.New("machine command arguments appear to contain secret material")
	}

	switch program {
	case "kubectl":
		return validateKubectlMachineCommand(req.Args)
	case "helm":
		return validateHelmMachineCommand(req.Args)
	case "systemctl":
		return validateSystemctlMachineCommand(req.Args)
	case "journalctl":
		return validateJournalctlMachineCommand(req.Args)
	case "docker":
		return validateDockerMachineCommand(req.Args)
	case "crictl":
		return validateCrictlMachineCommand(req.Args)
	case "df", "free", "uptime", "uname", "lsblk", "findmnt", "vmstat", "ss", "hostname":
		return validateDiagnosticMachineCommand(program, req.Args)
	default:
		return false, fmt.Errorf("machine command program %q is not allowlisted", req.Program)
	}
}

func validateKubectlMachineCommand(args []string) (bool, error) {
	if len(args) == 0 {
		return false, errors.New("kubectl verb is required")
	}
	if machineArgsContainFlag(args,
		"--kubeconfig", "--server", "--token", "--username", "--password",
		"--client-key", "--client-certificate", "--certificate-authority",
		"--proxy-url", "--context", "--as", "--as-group", "--raw",
		"--filename", "-f", "--watch", "-w",
	) {
		return false, errors.New("kubectl connection, credential, impersonation, raw-API, filename or watch overrides are not allowed")
	}
	if kubectlArgsReferenceSecret(args) {
		return false, errors.New("kubectl Secret resources are not exposed through direct machine commands")
	}

	verb := strings.ToLower(args[0])
	switch verb {
	case "get", "describe", "top", "version", "cluster-info", "api-resources", "api-versions", "explain", "events", "wait":
		return true, nil
	case "logs":
		if machineArgsContainFlag(args, "--follow", "-f") {
			return false, errors.New("kubectl logs follow mode is not allowed; request a bounded log window")
		}
		return true, nil
	case "config":
		if len(args) >= 2 && strings.EqualFold(args[1], "current-context") {
			return true, nil
		}
		return false, errors.New("only kubectl config current-context is allowed")
	case "auth":
		if len(args) >= 2 && strings.EqualFold(args[1], "can-i") {
			return true, nil
		}
		return false, errors.New("only kubectl auth can-i is allowed")
	case "rollout":
		if len(args) < 2 {
			return false, errors.New("kubectl rollout subcommand is required")
		}
		switch strings.ToLower(args[1]) {
		case "status", "history":
			return true, nil
		case "restart", "undo", "pause", "resume":
			return false, nil
		default:
			return false, errors.New("kubectl rollout subcommand is not allowlisted")
		}
	case "scale", "patch", "annotate", "label":
		return false, nil
	case "set":
		if len(args) < 2 {
			return false, errors.New("kubectl set subcommand is required")
		}
		switch strings.ToLower(args[1]) {
		case "image", "env", "resources":
			return false, nil
		default:
			return false, errors.New("kubectl set subcommand is not allowlisted")
		}
	default:
		return false, errors.New("kubectl verb is not allowlisted for direct execution")
	}
}

func validateHelmMachineCommand(args []string) (bool, error) {
	if len(args) == 0 {
		return false, errors.New("helm command is required")
	}
	if machineArgsContainFlag(args,
		"--kubeconfig", "--kube-context", "--repository-config", "--repository-cache",
		"--registry-config", "--post-renderer", "--post-renderer-args",
	) {
		return false, errors.New("helm connection, repository/registry or post-renderer overrides are not allowed")
	}
	switch strings.ToLower(args[0]) {
	case "version", "list", "status", "history", "env", "show":
		return true, nil
	case "upgrade", "install", "rollback":
		return false, nil
	default:
		return false, errors.New("helm command is not allowlisted for direct execution")
	}
}

func validateSystemctlMachineCommand(args []string) (bool, error) {
	if len(args) == 0 {
		return false, errors.New("systemctl command is required")
	}
	if machineArgsContainFlag(args, "--host", "-H", "--machine", "-M", "--root", "--image", "--force") {
		return false, errors.New("systemctl remote/root/image/force overrides are not allowed")
	}
	idx := 0
	for idx < len(args) && strings.HasPrefix(args[idx], "-") {
		if args[idx] != "--user" && args[idx] != "--no-pager" && args[idx] != "--no-legend" && args[idx] != "--plain" {
			return false, errors.New("systemctl global option is not allowlisted")
		}
		idx++
	}
	if idx >= len(args) {
		return false, errors.New("systemctl command is required")
	}
	switch strings.ToLower(args[idx]) {
	case "status", "is-active", "is-enabled", "is-failed", "list-units", "list-unit-files", "get-default":
		return true, nil
	case "restart", "start", "stop", "reload", "try-restart", "enable", "disable", "daemon-reload", "reset-failed":
		return false, nil
	default:
		return false, errors.New("systemctl command is not allowlisted for direct execution")
	}
}

func validateJournalctlMachineCommand(args []string) (bool, error) {
	if machineArgsContainFlag(args,
		"--vacuum-size", "--vacuum-time", "--vacuum-files", "--rotate", "--flush", "--sync", "--relinquish-var",
		"--file", "--directory", "--root", "--image", "--machine", "-M",
	) {
		return false, errors.New("journalctl mutation or alternate-root/file access is not allowed")
	}
	if machineArgsContainFlag(args, "-f", "--follow") {
		return false, errors.New("journalctl follow mode is not allowed; request a bounded log window instead")
	}
	return true, nil
}

func validateDockerMachineCommand(args []string) (bool, error) {
	if len(args) == 0 {
		return false, errors.New("docker command is required")
	}
	if machineArgsContainFlag(args, "--host", "-H", "--context", "--config", "--tls", "--tlsverify", "--tlscacert", "--tlscert", "--tlskey") {
		return false, errors.New("docker host/context/TLS overrides are not allowed")
	}
	switch strings.ToLower(args[0]) {
	case "ps", "images", "info", "version":
		return true, nil
	case "logs":
		if machineArgsContainFlag(args, "--follow", "-f") {
			return false, errors.New("docker logs follow mode is not allowed; request a bounded log window")
		}
		return true, nil
	case "stats":
		if !machineArgsContainFlag(args, "--no-stream") {
			return false, errors.New("docker stats requires --no-stream")
		}
		return true, nil
	case "network", "volume":
		if len(args) >= 2 && strings.EqualFold(args[1], "ls") {
			return true, nil
		}
		return false, errors.New("only docker network/volume ls is allowed")
	case "restart", "start", "stop", "pause", "unpause":
		return false, nil
	case "compose":
		if len(args) < 2 {
			return false, errors.New("docker compose subcommand is required")
		}
		switch strings.ToLower(args[1]) {
		case "ps", "images":
			return true, nil
		case "logs":
			if machineArgsContainFlag(args[2:], "--follow", "-f") {
				return false, errors.New("docker compose logs follow mode is not allowed; request a bounded log window")
			}
			return true, nil
		case "up", "restart", "start", "stop", "pull":
			return false, nil
		default:
			return false, errors.New("docker compose subcommand is not allowlisted")
		}
	default:
		return false, errors.New("docker command is not allowlisted for direct execution")
	}
}

func validateCrictlMachineCommand(args []string) (bool, error) {
	if len(args) == 0 {
		return false, errors.New("crictl command is required")
	}
	if machineArgsContainFlag(args, "--runtime-endpoint", "--image-endpoint", "--config", "--creds", "--auth") {
		return false, errors.New("crictl endpoint/config/credential overrides are not allowed")
	}
	switch strings.ToLower(args[0]) {
	case "ps", "pods", "images", "stats", "info", "version":
		return true, nil
	case "logs":
		if machineArgsContainFlag(args, "--follow", "-f") {
			return false, errors.New("crictl logs follow mode is not allowed; request a bounded log window")
		}
		return true, nil
	default:
		return false, errors.New("crictl command is not allowlisted for direct execution")
	}
}

func validateDiagnosticMachineCommand(program string, args []string) (bool, error) {
	if program == "hostname" {
		for _, arg := range args {
			switch arg {
			case "-f", "--fqdn", "-s", "--short", "-d", "--domain", "-i", "--ip-address", "-I", "--all-ip-addresses":
			default:
				return false, errors.New("hostname arguments are limited to read-only query flags")
			}
		}
	}
	if program == "ss" {
		for _, arg := range args {
			if arg == "-p" || arg == "--processes" || strings.Contains(arg, "p") && strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "--") {
				return false, errors.New("ss process inspection is not exposed through direct machine commands")
			}
		}
	}
	return true, nil
}

func machineArgsContainFlag(args []string, flags ...string) bool {
	for _, arg := range args {
		low := strings.ToLower(strings.TrimSpace(arg))
		for _, flag := range flags {
			f := strings.ToLower(flag)
			if low == f || strings.HasPrefix(low, f+"=") {
				return true
			}
			// Several CLIs accept a short option with its value attached, e.g.
			// docker -Htcp://host or systemctl -Hhost. Treat those exactly like
			// the separate-argv form when the short flag itself is blocked.
			if len(f) == 2 && strings.HasPrefix(f, "-") && !strings.HasPrefix(f, "--") && strings.HasPrefix(low, f) && len(low) > len(f) {
				return true
			}
		}
	}
	return false
}

func kubectlArgsReferenceSecret(args []string) bool {
	for _, arg := range args {
		for _, token := range strings.Split(strings.ToLower(strings.TrimSpace(arg)), ",") {
			resource := token
			if slash := strings.Index(resource, "/"); slash >= 0 {
				resource = resource[:slash]
			}
			if dot := strings.Index(resource, "."); dot >= 0 {
				resource = resource[:dot]
			}
			if resource == "secret" || resource == "secrets" {
				return true
			}
		}
	}
	return false
}

type machineInvocationResult struct {
	output    string
	truncated bool
	exitCode  int
	err       error
}

func runMachineInvocation(ctx context.Context, name string, args []string) machineInvocationResult {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = append(os.Environ(),
		"PAGER=cat",
		"GIT_PAGER=cat",
		"SYSTEMD_PAGER=cat",
		"SYSTEMD_COLORS=0",
		"NO_COLOR=1",
	)
	configureChildProcess(cmd, false)
	output := &limitedCapture{limit: maxMachineCommandOutputBytes}
	cmd.Stdout, cmd.Stderr = output, output
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}
	return machineInvocationResult{
		output:    strings.TrimSpace(output.String()),
		truncated: output.exceeded,
		exitCode:  exitCode,
		err:       err,
	}
}

func isK3sKubeconfigPermissionFailure(out string, err error) bool {
	if err == nil {
		return false
	}
	low := strings.ToLower(out + " " + err.Error())
	if !strings.Contains(low, "/etc/rancher/k3s/k3s.yaml") {
		return false
	}
	return strings.Contains(low, "permission denied") || strings.Contains(low, "unable to read")
}

func executeMachineCommand(ctx context.Context, req MachineCommandRequest, readOnly bool) (MachineCommandResult, error) {
	program := strings.ToLower(strings.TrimSpace(req.Program))
	args := append([]string(nil), req.Args...)
	result := MachineCommandResult{Program: program, Args: args, ReadOnly: readOnly, ExitCode: 0, Transport: "direct"}

	timeout := defaultMachineCommandTimeout
	if req.TimeoutSeconds > 0 {
		timeout = time.Duration(req.TimeoutSeconds) * time.Second
	}
	if timeout > maxMachineCommandTimeout {
		timeout = maxMachineCommandTimeout
	}
	if timeout < time.Second {
		timeout = time.Second
	}
	commandCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	attempt := runMachineInvocation(commandCtx, program, args)
	if LooksSecret(attempt.output) {
		result.Output = "[withheld by Workbench: machine command output resembled secret material]"
		result.ExitCode = attempt.exitCode
		return result, errors.New("machine command output was withheld because it resembled secret material")
	}

	// DaisyClover's K3s host deliberately keeps /etc/rancher/k3s/k3s.yaml
	// root-readable and its existing cluster runbooks use non-interactive sudo
	// for kubectl. Keep that privilege boundary intact: only after the request has
	// passed Workbench's strict kubectl policy, and only when direct kubectl fails
	// specifically on the K3s kubeconfig permission boundary, retry the exact argv
	// through `sudo -n k3s kubectl`. `sudo` is never exposed as a Workbench
	// program and no shell text is evaluated.
	if program == "kubectl" && isK3sKubeconfigPermissionFailure(attempt.output, attempt.err) && commandCtx.Err() == nil {
		elevatedArgs := append([]string{"-n", "k3s", "kubectl"}, args...)
		attempt = runMachineInvocation(commandCtx, "sudo", elevatedArgs)
		result.Transport = "k3s-sudo"
		if LooksSecret(attempt.output) {
			result.Output = "[withheld by Workbench: machine command output resembled secret material]"
			result.ExitCode = attempt.exitCode
			return result, errors.New("machine command output was withheld because it resembled secret material")
		}
	}

	out := attempt.output
	result.Truncated = attempt.truncated
	if attempt.truncated {
		out += "\n… output truncated by Workbench …"
	}
	result.Output = out
	result.ExitCode = attempt.exitCode

	if attempt.err != nil {
		if errors.Is(commandCtx.Err(), context.DeadlineExceeded) {
			return result, errors.New("machine command timed out")
		}
		return result, fmt.Errorf("machine command failed: %w", attempt.err)
	}
	return result, nil
}
