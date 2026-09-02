package core

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestMachineCommandPolicySeparatesReadAndMutation(t *testing.T) {
	readOnly, err := validateMachineCommand(MachineCommandRequest{Program: "kubectl", Args: []string{"get", "pods", "-A"}})
	if err != nil || !readOnly {
		t.Fatalf("kubectl get should be read-only: readOnly=%t err=%v", readOnly, err)
	}
	readOnly, err = validateMachineCommand(MachineCommandRequest{Program: "kubectl", Args: []string{"rollout", "restart", "deployment/web", "-n", "app-dev"}})
	if err != nil || readOnly {
		t.Fatalf("kubectl rollout restart should be mutating: readOnly=%t err=%v", readOnly, err)
	}
	readOnly, err = validateMachineCommand(MachineCommandRequest{Program: "systemctl", Args: []string{"--user", "status", "workbench-mcp.service"}})
	if err != nil || !readOnly {
		t.Fatalf("systemctl status should be read-only: readOnly=%t err=%v", readOnly, err)
	}
	readOnly, err = validateMachineCommand(MachineCommandRequest{Program: "systemctl", Args: []string{"--user", "restart", "workbench-mcp.service"}})
	if err != nil || readOnly {
		t.Fatalf("systemctl restart should be mutating: readOnly=%t err=%v", readOnly, err)
	}
}

func TestRoutineMachineOperationsRemainOnDirectWorkbenchPolicy(t *testing.T) {
	cases := []struct {
		name     string
		req      MachineCommandRequest
		readOnly bool
	}{
		{name: "kubernetes inspection", req: MachineCommandRequest{Program: "kubectl", Args: []string{"get", "nodes", "-o", "wide"}}, readOnly: true},
		{name: "kubernetes mutation", req: MachineCommandRequest{Program: "kubectl", Args: []string{"rollout", "restart", "deployment/web", "-n", "app-dev"}}, readOnly: false},
		{name: "docker inspection", req: MachineCommandRequest{Program: "docker", Args: []string{"ps"}}, readOnly: true},
		{name: "docker allowed mutation", req: MachineCommandRequest{Program: "docker", Args: []string{"restart", "web"}}, readOnly: false},
		{name: "systemd inspection", req: MachineCommandRequest{Program: "systemctl", Args: []string{"--user", "status", "workbench-mcp.service"}}, readOnly: true},
		{name: "systemd allowed mutation", req: MachineCommandRequest{Program: "systemctl", Args: []string{"--user", "restart", "workbench-mcp.service"}}, readOnly: false},
		{name: "supported host inspection", req: MachineCommandRequest{Program: "uptime"}, readOnly: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			readOnly, err := validateMachineCommand(tc.req)
			if err != nil {
				t.Fatalf("routine direct Workbench operation was rejected: %v", err)
			}
			if readOnly != tc.readOnly {
				t.Fatalf("readOnly=%t, want %t for %+v", readOnly, tc.readOnly, tc.req)
			}
		})
	}
}

func TestMachineCommandPolicyRejectsShellsPivotsSecretsAndDestructiveVerbs(t *testing.T) {
	cases := []MachineCommandRequest{
		{Program: "bash", Args: []string{"-lc", "kubectl get pods"}},
		{Program: "/usr/bin/kubectl", Args: []string{"get", "pods"}},
		{Program: "kubectl", Args: []string{"get", "secrets", "-A"}},
		{Program: "kubectl", Args: []string{"get", "secret.v1", "example"}},
		{Program: "kubectl", Args: []string{"get", "pods,secrets", "-A"}},
		{Program: "kubectl", Args: []string{"delete", "pod", "web"}},
		{Program: "kubectl", Args: []string{"get", "pods", "--kubeconfig=/tmp/other"}},
		{Program: "kubectl", Args: []string{"get", "pods", "--server", "https://other.invalid"}},
		{Program: "kubectl", Args: []string{"get", "-f", "/tmp/object.yaml"}},
		{Program: "helm", Args: []string{"uninstall", "prod"}},
		{Program: "helm", Args: []string{"upgrade", "web", "./chart", "--post-renderer", "/tmp/script"}},
		{Program: "systemctl", Args: []string{"reboot"}},
		{Program: "systemctl", Args: []string{"-Hremote.example", "status", "ssh.service"}},
		{Program: "journalctl", Args: []string{"--vacuum-time=1d"}},
		{Program: "docker", Args: []string{"-Htcp://other:2375", "ps"}},
		{Program: "docker", Args: []string{"exec", "web", "sh"}},
		{Program: "docker", Args: []string{"rm", "web"}},
		{Program: "crictl", Args: []string{"rm", "deadbeef"}},
	}
	for _, req := range cases {
		if readOnly, err := validateMachineCommand(req); err == nil {
			t.Fatalf("unsafe machine request accepted readOnly=%t: %+v", readOnly, req)
		}
	}
}

func TestMachineCommandPolicyRequiresBoundedReadForms(t *testing.T) {
	if _, err := validateMachineCommand(MachineCommandRequest{Program: "docker", Args: []string{"stats"}}); err == nil {
		t.Fatal("docker stats without --no-stream must be rejected")
	}
	if readOnly, err := validateMachineCommand(MachineCommandRequest{Program: "docker", Args: []string{"stats", "--no-stream"}}); err != nil || !readOnly {
		t.Fatalf("bounded docker stats rejected: readOnly=%t err=%v", readOnly, err)
	}
	for _, req := range []MachineCommandRequest{
		{Program: "journalctl", Args: []string{"-f", "-u", "workbench-mcp"}},
		{Program: "kubectl", Args: []string{"get", "pods", "-w"}},
		{Program: "kubectl", Args: []string{"logs", "-f", "pod/web"}},
		{Program: "docker", Args: []string{"logs", "--follow", "web"}},
		{Program: "docker", Args: []string{"compose", "logs", "-f"}},
		{Program: "crictl", Args: []string{"logs", "-f", "deadbeef"}},
	} {
		if _, err := validateMachineCommand(req); err == nil {
			t.Fatalf("unbounded read form accepted: %+v", req)
		}
	}
	if _, err := validateMachineCommand(MachineCommandRequest{Program: "hostname", Args: []string{"new-hostname"}}); err == nil {
		t.Fatal("hostname mutation must be rejected")
	}
}

func TestInspectAndMutateToolsRejectWrongCommandClass(t *testing.T) {
	if _, err := InspectMachine(context.Background(), MachineCommandRequest{Program: "systemctl", Args: []string{"restart", "example.service"}}); err == nil || !strings.Contains(err.Error(), "mutating") {
		t.Fatalf("inspect_machine should reject mutation, got %v", err)
	}
	if _, err := RunMachineCommand(context.Background(), MachineCommandRequest{Program: "uptime"}); err == nil || !strings.Contains(err.Error(), "read-only") {
		t.Fatalf("run_machine_command should reject read-only command, got %v", err)
	}
}

func TestInspectMachineExecutesArgvWithoutShell(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX fake executable smoke")
	}
	bin := t.TempDir()
	path := filepath.Join(bin, "kubectl")
	script := "#!/bin/sh\nprintf 'argc=%s\\n' \"$#\"\nprintf 'arg1=%s\\n' \"$1\"\nprintf 'arg2=%s\\n' \"$2\"\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	result, err := InspectMachine(context.Background(), MachineCommandRequest{Program: "kubectl", Args: []string{"get", "pods;echo-not-a-shell"}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Output, "argc=2") || !strings.Contains(result.Output, "arg2=pods;echo-not-a-shell") {
		t.Fatalf("argv was not preserved literally: %q", result.Output)
	}
}
