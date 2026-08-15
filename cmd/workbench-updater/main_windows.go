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
	mbOK              = 0x00000000
	mbYesNo           = 0x00000004
	mbIconError       = 0x00000010
	mbIconInformation = 0x00000040
	mbIconWarning     = 0x00000030
	idYes             = 6
)

var (
	updaterUser32       = syscall.NewLazyDLL("user32.dll")
	updaterMessageBoxW  = updaterUser32.NewProc("MessageBoxW")
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
	release, err := core.FetchOfficialLatestRelease(ctx)
	if err != nil {
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
	if err := tx.Commit(); err != nil {
		updaterMessage("Workbench updated", "Workbench v"+release.Version+" was installed and launched successfully.\n\nThe old executable backup could not be removed automatically; the new verified app remains installed.", mbIconWarning)
		return nil
	}
	updaterMessage("Workbench updated", "Workbench v"+release.Version+" was installed from the verified official release and launched successfully.", mbIconInformation)
	return nil
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
