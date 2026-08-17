//go:build windows

package core

import (
	"errors"
	"fmt"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

var (
	runnerShell32          = syscall.NewLazyDLL("shell32.dll")
	runnerProcShellExecute = runnerShell32.NewProc("ShellExecuteW")
)

type runnerSSHConsoleSpec struct {
	File       string
	Parameters string
}

func runRunnerSSHConsole(host string, remoteArgs []string) error {
	return launchRunnerSSHConsole(host, remoteArgs)
}

func startRunnerSSHConsole(host string, remoteArgs []string) error {
	spec, err := runnerSSHConsoleLaunchSpec(host, remoteArgs)
	if err != nil {
		return err
	}
	// The Settings handler currently shows its explanatory modal immediately
	// after this call returns. Launching the console synchronously lets that modal
	// steal foreground focus, making a correctly-created console look as if it
	// never opened. Queue the operator-visible console just after the modal is on
	// screen; Windows can then foreground the newly launched console. /K keeps it
	// present after SSH exits, so the operator can read any authentication error.
	go func() {
		time.Sleep(450 * time.Millisecond)
		_ = shellExecuteRunnerSSHConsole(spec)
	}()
	return nil
}

func launchRunnerSSHConsole(host string, remoteArgs []string) error {
	spec, err := runnerSSHConsoleLaunchSpec(host, remoteArgs)
	if err != nil {
		return err
	}
	return shellExecuteRunnerSSHConsole(spec)
}

func runnerSSHConsoleLaunchSpec(host string, remoteArgs []string) (runnerSSHConsoleSpec, error) {
	parts := []string{
		"ssh",
		"-t",
		"-o", "ConnectTimeout=10",
		"-o", "StrictHostKeyChecking=accept-new",
		host,
	}
	parts = append(parts, remoteArgs...)
	for _, part := range parts {
		if !runnerSSHConsoleTokenSafe(part) {
			return runnerSSHConsoleSpec{}, errors.New("runner SSH console command contains an unsafe token")
		}
	}
	return runnerSSHConsoleSpec{
		File:       "cmd.exe",
		Parameters: "/D /K " + strings.Join(parts, " "),
	}, nil
}

func shellExecuteRunnerSSHConsole(spec runnerSSHConsoleSpec) error {
	verb, err := syscall.UTF16PtrFromString("open")
	if err != nil {
		return err
	}
	file, err := syscall.UTF16PtrFromString(spec.File)
	if err != nil {
		return err
	}
	params, err := syscall.UTF16PtrFromString(spec.Parameters)
	if err != nil {
		return err
	}

	// Workbench.exe is built as a Windows GUI binary and therefore does not own
	// console standard handles. os/exec + CREATE_NEW_CONSOLE can create a window
	// while still handing cmd.exe NUL stdin/stdout/stderr, which makes an SSH
	// login prompt effectively invisible and unusable. ShellExecute lets Windows
	// start the console-subsystem cmd.exe normally, giving it real CONIN$/CONOUT$
	// handles. /K intentionally keeps the window visible if SSH exits with an
	// error so the operator can read the diagnostic instead of seeing a flash.
	ret, _, callErr := runnerProcShellExecute.Call(
		0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(file)),
		uintptr(unsafe.Pointer(params)),
		0,
		1, // SW_SHOWNORMAL
	)
	if ret <= 32 {
		return fmt.Errorf("Windows could not open the runner SSH console (ShellExecuteW=%d): %v", ret, callErr)
	}
	return nil
}

func runnerSSHConsoleTokenSafe(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			continue
		}
		switch r {
		case '@', '.', '_', '-', ':', '/', '[', ']', '$', '=':
			continue
		default:
			return false
		}
	}
	return true
}
