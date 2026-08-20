//go:build windows

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

const (
	mbOK               = 0x00000000
	mbYesNo            = 0x00000004
	mbIconError        = 0x00000010
	mbIconInformation  = 0x00000040
	mbIconWarning      = 0x00000030
	idYes              = 6
	wmClose            = 0x0010
	processSynchronize = 0x00100000
	waitObject0        = 0x00000000
	waitTimeout        = 0x00000102
	waitFailed         = 0xffffffff
)

var workbenchWindowClasses = []string{
	"DaisyCloverWorkbenchProductionDashboard",
	"DaisyCloverWorkbenchProductionWindow",
}

var (
	updaterUser32                   = syscall.NewLazyDLL("user32.dll")
	updaterKernel32                 = syscall.NewLazyDLL("kernel32.dll")
	updaterMessageBoxW              = updaterUser32.NewProc("MessageBoxW")
	updaterFindWindowW              = updaterUser32.NewProc("FindWindowW")
	updaterGetWindowThreadProcessID = updaterUser32.NewProc("GetWindowThreadProcessId")
	updaterPostMessageW             = updaterUser32.NewProc("PostMessageW")
	updaterOpenProcess              = updaterKernel32.NewProc("OpenProcess")
	updaterWaitForSingleObject      = updaterKernel32.NewProc("WaitForSingleObject")
	updaterCloseHandle              = updaterKernel32.NewProc("CloseHandle")
)

func main() {
	if err := runUpdater(); err != nil {
		updaterMessage("Workbench update failed", err.Error(), mbIconError)
		os.Exit(1)
	}
}

func runUpdater() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, err = filepath.Abs(exe)
	if err != nil {
		return err
	}
	target := filepath.Join(filepath.Dir(exe), core.WindowsReleaseAsset)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	release, err := core.FetchOfficialLatestReleaseResilient(ctx)
	if err != nil {
		if core.IsUpdateCheckUnavailable(err) {
			updaterMessage(
				"Workbench update check unavailable",
				"Workbench could not reach GitHub after several attempts. Your current Workbench installation has not been changed.\n\nThis is usually a temporary network or DNS problem; try the update check again later.\n\nDetails:\n"+err.Error(),
				mbIconWarning,
			)
			return nil
		}
		return fmt.Errorf("could not check the official Workbench release: %w", err)
	}
	asset, err := core.DownloadVerifiedReleaseAsset(ctx, release, core.WindowsReleaseAsset, core.WindowsReleaseChecksumAsset, 128<<20)
	if err != nil {
		return fmt.Errorf("could not verify the official Workbench download: %w", err)
	}
	defer os.RemoveAll(filepath.Dir(asset.Path))
	if err := core.VerifyWindowsAMD64PE(asset.Path); err != nil {
		return fmt.Errorf("official Workbench download failed Windows executable validation: %w", err)
	}

	matches, err := core.WindowsAppMatchesVerifiedAsset(target, asset)
	if err == nil && matches {
		updaterMessage("Workbench is current", "Workbench.exe already matches the latest verified stable release (v"+release.Version+").", mbIconInformation)
		return nil
	}
	if err != nil && !os.IsNotExist(err) {
		// A damaged/non-PE existing target is still replaceable as long as it is
		// a regular file; BeginWindowsAppInstall performs the final safety check.
		if info, statErr := os.Lstat(target); statErr != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return fmt.Errorf("cannot inspect existing Workbench.exe: %w", err)
		}
	}

	action := "Install"
	if _, err := os.Stat(target); err == nil {
		action = "Update"
	}
	question := action + " Workbench.exe to the latest verified stable release (v" + release.Version + ")?\n\n" +
		"Location:\n" + target + "\n\n" +
		"Only the official DaisyCloverSoftware/workbench GitHub release and its published SHA-256 checksums will be used."
	if updaterMessage(action+" Workbench", question, mbYesNo|mbIconInformation) != idYes {
		return nil
	}

	// Workbench owns a per-user single-instance mutex. Launching the replacement
	// before the old desktop exits makes the new process immediately terminate as
	// "already running". Close the exact Workbench top-level window only after the
	// user accepted the verified update, then wait for its process (and embedded
	// MCP listener/mutex) to finish before swapping and relaunching the executable.
	if err := closeRunningWorkbenchForUpdate(); err != nil {
		return err
	}

	tx, err := core.BeginWindowsAppInstall(asset, target)
	if err != nil {
		return err
	}
	cmd := exec.Command(target)
	cmd.Dir = filepath.Dir(target)
	if err := cmd.Start(); err != nil {
		rollbackErr := tx.Rollback()
		return errorsJoin(fmt.Errorf("could not launch updated Workbench.exe: %w", err), rollbackErr)
	}
	if cmd.Process != nil {
		_ = cmd.Process.Release()
	}
	if err := tx.Commit(); err != nil {
		updaterMessage("Workbench updated", "Workbench v"+release.Version+" was installed and launched successfully.\n\nThe old executable backup could not be removed automatically; the new verified app remains installed.", mbIconWarning)
		return nil
	}

	// A successful relaunch is its own visible confirmation. Exit immediately so
	// the new Workbench can refresh this separately shipped updater executable if
	// the release also contains an updater fix; do not keep the updater locked
	// behind a modal success dialog.
	return nil
}

func closeRunningWorkbenchForUpdate() error {
	hwnd, pid := runningWorkbenchWindow()
	if hwnd == 0 || pid == 0 {
		return nil
	}

	process, _, openErr := updaterOpenProcess.Call(processSynchronize, 0, uintptr(pid))
	if process == 0 {
		return fmt.Errorf("could not open the running Workbench process for update handoff: %v", openErr)
	}
	defer updaterCloseHandle.Call(process)

	if posted, _, postErr := updaterPostMessageW.Call(hwnd, wmClose, 0, 0); posted == 0 {
		return fmt.Errorf("could not ask the running Workbench window to close for update: %v", postErr)
	}

	waitResult, _, waitErr := updaterWaitForSingleObject.Call(process, 20_000)
	switch waitResult {
	case waitObject0:
		// Process exit closes the single-instance mutex and embedded MCP listener.
		return nil
	case waitTimeout:
		return fmt.Errorf("Workbench did not close within 20 seconds; close it manually and retry the update")
	case waitFailed:
		return fmt.Errorf("could not wait for Workbench to close for update: %v", waitErr)
	default:
		return fmt.Errorf("unexpected Workbench update handoff wait result %#x", waitResult)
	}
}

func runningWorkbenchWindow() (uintptr, uint32) {
	for _, className := range workbenchWindowClasses {
		classPtr, _ := syscall.UTF16PtrFromString(className)
		hwnd, _, _ := updaterFindWindowW.Call(uintptr(unsafe.Pointer(classPtr)), 0)
		if hwnd == 0 {
			continue
		}
		var pid uint32
		updaterGetWindowThreadProcessID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
		if pid != 0 {
			return hwnd, pid
		}
	}
	return 0, 0
}

func updaterMessage(title, text string, flags uintptr) int {
	titlePtr, _ := syscall.UTF16PtrFromString(title)
	textPtr, _ := syscall.UTF16PtrFromString(text)
	r, _, _ := updaterMessageBoxW.Call(0, uintptr(unsafe.Pointer(textPtr)), uintptr(unsafe.Pointer(titlePtr)), flags)
	return int(r)
}

func errorsJoin(primary, secondary error) error {
	if secondary == nil {
		return primary
	}
	return fmt.Errorf("%v; rollback also failed: %w", primary, secondary)
}
