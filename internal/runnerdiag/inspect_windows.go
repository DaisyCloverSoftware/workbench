//go:build windows

package runnerdiag

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"
)

var advapi = syscall.NewLazyDLL("advapi32.dll")
var openSCManager = advapi.NewProc("OpenSCManagerW")
var enumServices = advapi.NewProc("EnumServicesStatusExW")
var openService = advapi.NewProc("OpenServiceW")
var queryServiceConfig = advapi.NewProc("QueryServiceConfigW")
var closeServiceHandle = advapi.NewProc("CloseServiceHandle")
var kernel = syscall.NewLazyDLL("kernel32.dll")
var queryProcessImage = kernel.NewProc("QueryFullProcessImageNameW")
var getSystemDirectory = kernel.NewProc("GetSystemDirectoryW")
var getDriveType = kernel.NewProc("GetDriveTypeW")
var getFinalPath = kernel.NewProc("GetFinalPathNameByHandleW")

type serviceStatusProcess struct{ Type, State, Controls, Win32Exit, SpecificExit, Checkpoint, WaitHint, PID, Flags uint32 }
type enumServiceRecord struct {
	Name, Display *uint16
	Status        serviceStatusProcess
}
type serviceConfig struct {
	Type, Start, Error             uint32
	Binary, Group                  *uint16
	Tag                            uint32
	Dependencies, Account, Display *uint16
}
type candidate struct{ root, source string }

// Inspect runs locally with query-only Windows handles. It starts no process,
// reads no credentials/logs/environment, and never traverses arbitrary trees.
func Inspect() (Report, error) {
	r := newReport()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	roots := []candidate{}
	add := func(root, source string) string {
		if !safeAbsolute(root) {
			return ""
		}
		root = filepath.Clean(root)
		id := opaqueID("runner", root)
		for _, c := range roots {
			if strings.EqualFold(c.root, root) {
				return id
			}
		}
		if len(roots) >= MaxInstallations {
			r.CandidateLimitReached = true
			return ""
		}
		roots = append(roots, candidate{root, source})
		return id
	}
	observeServices(ctx, &r, add)
	observeProcesses(ctx, &r, add)
	// Conventional locations only, not a claim to search all disks/user profiles.
	var system [260]uint16
	n, _, _ := getSystemDirectory.Call(uintptr(unsafe.Pointer(&system[0])), uintptr(len(system)))
	if n > 0 && n < uintptr(len(system)) {
		add(filepath.Join(filepath.VolumeName(syscall.UTF16ToString(system[:])), `\actions-runner`), "system_drive_conventional")
	}
	if home, err := os.UserHomeDir(); err == nil {
		add(filepath.Join(home, "actions-runner"), "current_user_conventional")
		add(filepath.Join(home, "work", "actions-runner"), "current_user_work_conventional")
	}
	for _, c := range roots {
		if ctx.Err() != nil {
			r.LocationsUnreadable += len(roots) - r.LocationsChecked
			break
		}
		r.LocationsChecked++
		install, status := inspectRoot(c)
		r.Locations = append(r.Locations, Location{ID: opaqueID("runner", c.root), Source: c.source, ReadStatus: status})
		if status == "absent" {
			r.LocationsAbsent++
			continue
		}
		if status != "ok" {
			r.LocationsUnreadable++
			continue
		}
		r.Installations = append(r.Installations, install)
	}
	return r, nil
}
func InspectJSON() (string, error) {
	r, e := Inspect()
	if e != nil {
		return "", e
	}
	return encodeReport(r)
}
func safeAbsolute(p string) bool {
	// Only ordinary local drive paths. No UNC/device paths, ADS or traversal.
	if len(p) < 3 || len(p) > 1024 || p[1] != ':' || (p[2] != '\\' && p[2] != '/') || !((p[0] >= 'A' && p[0] <= 'Z') || (p[0] >= 'a' && p[0] <= 'z')) {
		return false
	}
	if strings.ContainsAny(p[2:], ":\x00\r\n%") {
		return false
	}
	for _, s := range strings.Split(strings.ReplaceAll(p, `\`, "/"), "/")[1:] {
		if s == ".." || s == "." || strings.EqualFold(s, ".git") || strings.HasSuffix(s, " ") || strings.HasSuffix(s, ".") {
			return false
		}
	}
	return filepath.IsAbs(p)
}
func installationFromImage(image string) string {
	p := strings.TrimSpace(image)
	if strings.HasPrefix(p, `"`) {
		if !strings.HasSuffix(p, `"`) || len(p) < 2 {
			return ""
		}
		p = p[1 : len(p)-1]
	}
	if strings.Contains(p, `"`) || !safeAbsolute(p) {
		return ""
	}
	if !strings.EqualFold(filepath.Base(filepath.Dir(p)), "bin") {
		return ""
	}
	switch strings.ToLower(filepath.Base(p)) {
	case "runnerservice.exe", "runner.listener.exe", "runner.worker.exe":
		return filepath.Dir(filepath.Dir(p))
	}
	return ""
}
func nativeString(buf []byte, p *uint16) (string, bool) {
	if p == nil || len(buf) < 2 {
		return "", false
	}
	base := uintptr(unsafe.Pointer(&buf[0]))
	ptr := uintptr(unsafe.Pointer(p))
	if ptr < base || ptr-base >= uintptr(len(buf)) || (ptr-base)%2 != 0 {
		return "", false
	}
	start := int(ptr - base)
	words := make([]uint16, 0, 128)
	for off := start; off+1 < len(buf) && len(words) < 4096; off += 2 {
		w := binary.LittleEndian.Uint16(buf[off : off+2])
		if w == 0 {
			return string(utf16.Decode(words)), true
		}
		words = append(words, w)
	}
	return "", false
}
func stateName(v uint32) string {
	switch v {
	case 1:
		return "stopped"
	case 2:
		return "start_pending"
	case 3:
		return "stop_pending"
	case 4:
		return "running"
	case 5:
		return "continue_pending"
	case 6:
		return "pause_pending"
	case 7:
		return "paused"
	}
	return "unknown"
}
func modeName(v uint32) string {
	switch v {
	case 2:
		return "automatic"
	case 3:
		return "manual"
	case 4:
		return "disabled"
	}
	return "other"
}
func failure(err error) string {
	if errors.Is(err, syscall.ERROR_ACCESS_DENIED) {
		return "access_denied"
	}
	return "query_failed"
}
func observeServices(ctx context.Context, r *Report, add func(string, string) string) {
	manager, _, err := openSCManager.Call(0, 0, 0x0005) // CONNECT | ENUMERATE_SERVICE, no mutation rights.
	if manager == 0 {
		r.ServicesRead = failure(err)
		return
	}
	defer closeServiceHandle.Call(manager)
	var resume uint32
	r.ServicesRead = "visible_services_only" // Windows may silently omit inaccessible services.
	for page := 0; page < 4; page++ {
		if ctx.Err() != nil {
			r.ServicesRead = "deadline"
			return
		}
		buf := make([]byte, 256<<10)
		var needed, count uint32
		ok, _, e := enumServices.Call(manager, 0, 0x30, 3, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)), uintptr(unsafe.Pointer(&needed)), uintptr(unsafe.Pointer(&count)), uintptr(unsafe.Pointer(&resume)), 0)
		size := unsafe.Sizeof(enumServiceRecord{})
		if uintptr(count) > uintptr(len(buf))/size {
			r.ServicesRead = "invalid_native_result"
			return
		}
		entries := unsafe.Slice((*enumServiceRecord)(unsafe.Pointer(&buf[0])), int(count))
		for _, entry := range entries {
			name, valid := nativeString(buf, entry.Name)
			if !valid || !strings.HasPrefix(strings.ToLower(name), "actions.runner.") {
				continue
			}
			if len(r.Services) >= MaxRecords {
				r.ServicesRead = "record_limit"
				return
			}
			s := Service{ID: opaqueID("service", name), State: stateName(entry.Status.State), StartMode: "unknown", ConfigRead: "not_observed"}
			handle, _, openErr := openService.Call(manager, uintptr(unsafe.Pointer(entry.Name)), 0x0001) // SERVICE_QUERY_CONFIG only.
			if handle == 0 {
				s.ConfigRead = failure(openErr)
			} else {
				raw := make([]byte, 8192)
				var bytesNeeded uint32
				good, _, readErr := queryServiceConfig.Call(handle, uintptr(unsafe.Pointer(&raw[0])), uintptr(len(raw)), uintptr(unsafe.Pointer(&bytesNeeded)))
				if good == 0 {
					s.ConfigRead = failure(readErr)
				} else {
					cfg := (*serviceConfig)(unsafe.Pointer(&raw[0]))
					image, readable := nativeString(raw, cfg.Binary)
					s.StartMode = modeName(cfg.Start)
					s.ConfigRead = "readable"
					if !readable {
						s.ConfigRead = "invalid_image"
					} else if root := installationFromImage(image); root != "" {
						s.InstallationID = add(root, "registered_service")
					} else {
						s.ConfigRead = "nonstandard_image_not_followed"
					}
				}
				closeServiceHandle.Call(handle)
			}
			r.Services = append(r.Services, s)
		}
		if ok != 0 {
			return
		}
		if e != syscall.ERROR_MORE_DATA {
			r.ServicesRead = failure(e)
			return
		}
	}
	r.ServicesRead = "page_limit"
}
func observeProcesses(ctx context.Context, r *Report, add func(string, string) string) {
	h, err := syscall.CreateToolhelp32Snapshot(0x00000002, 0)
	if err != nil {
		r.ProcessesRead = failure(err)
		return
	}
	defer syscall.CloseHandle(h)
	entry := syscall.ProcessEntry32{Size: uint32(unsafe.Sizeof(syscall.ProcessEntry32{}))}
	err = syscall.Process32First(h, &entry)
	r.ProcessesRead = "snapshot_only"
	count := 0
	for err == nil {
		count++
		if count > 16384 || ctx.Err() != nil {
			r.ProcessesRead = "record_or_time_limit"
			return
		}
		name := strings.ToLower(syscall.UTF16ToString(entry.ExeFile[:]))
		if name == "runner.listener.exe" || name == "runner.worker.exe" || name == "runnerservice.exe" {
			if len(r.Processes) >= MaxRecords {
				r.ProcessesRead = "record_limit"
				return
			}
			p := Process{Kind: name, ImageRead: "not_observed"}
			proc, e := syscall.OpenProcess(0x1000, false, entry.ProcessID) // PROCESS_QUERY_LIMITED_INFORMATION only.
			if e != nil {
				p.ImageRead = failure(e)
			} else {
				var buf [2048]uint16
				size := uint32(len(buf))
				good, _, imageErr := queryProcessImage.Call(uintptr(proc), 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
				if good == 0 || size >= uint32(len(buf)) {
					p.ImageRead = failure(imageErr)
				} else {
					p.ImageRead = "readable"
					root := installationFromImage(syscall.UTF16ToString(buf[:size]))
					if root != "" {
						p.InstallationID = add(root, "running_process")
					} else {
						p.ImageRead = "nonstandard_image_not_followed"
					}
				}
				syscall.CloseHandle(proc)
			}
			r.Processes = append(r.Processes, p)
		}
		err = syscall.Process32Next(h, &entry)
	}
	if !errors.Is(err, syscall.ERROR_NO_MORE_FILES) {
		r.ProcessesRead = failure(err)
	}
}

// Lock directory components against rename while reading a fixed child. Reject
// all reparse points, hard-linked metadata and Git worktrees. No caller path.
func lockRoot(root string) ([]syscall.Handle, string) {
	if !safeAbsolute(root) {
		return nil, "unsafe_path"
	}
	volume, _ := syscall.UTF16PtrFromString(filepath.VolumeName(root) + `\`)
	dtype, _, _ := getDriveType.Call(uintptr(unsafe.Pointer(volume)))
	if dtype != 3 {
		return nil, "non_fixed_local_drive_excluded"
	}
	var handles []syscall.Handle
	release := func() {
		for _, h := range handles {
			syscall.CloseHandle(h)
		}
	}
	current := filepath.VolumeName(root) + `\`
	parts := strings.Split(strings.TrimPrefix(filepath.Clean(root), current), `\`)
	paths := []string{current}
	for _, p := range parts {
		if p == "" {
			continue
		}
		current = filepath.Join(current, p)
		paths = append(paths, current)
	}
	for _, p := range paths {
		ptr, e := syscall.UTF16PtrFromString(p)
		if e != nil {
			release()
			return nil, "unsafe_path"
		}
		h, e := syscall.CreateFile(ptr, 0x80, syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE, nil, syscall.OPEN_EXISTING, syscall.FILE_FLAG_OPEN_REPARSE_POINT|syscall.FILE_FLAG_BACKUP_SEMANTICS, 0)
		if e != nil {
			release()
			if errors.Is(e, syscall.ERROR_FILE_NOT_FOUND) || errors.Is(e, syscall.ERROR_PATH_NOT_FOUND) {
				return nil, "absent"
			}
			return nil, failure(e)
		}
		handles = append(handles, h)
		var info syscall.ByHandleFileInformation
		if syscall.GetFileInformationByHandle(h, &info) != nil || info.FileAttributes&syscall.FILE_ATTRIBUTE_REPARSE_POINT != 0 || info.FileAttributes&syscall.FILE_ATTRIBUTE_DIRECTORY == 0 {
			release()
			return nil, "unsafe_directory"
		}
		var final [2048]uint16
		n, _, _ := getFinalPath.Call(uintptr(h), uintptr(unsafe.Pointer(&final[0])), uintptr(len(final)), 0)
		if n == 0 || n >= uintptr(len(final)) || !strings.EqualFold(filepath.Clean(strings.TrimPrefix(syscall.UTF16ToString(final[:n]), `\\?\`)), filepath.Clean(p)) {
			release()
			return nil, "path_alias_or_unreadable_excluded"
		}
		// Metadata-only worktree exclusion; never open Git/source files.
		if _, e := os.Lstat(filepath.Join(p, ".git")); e == nil {
			release()
			return nil, "git_worktree_excluded"
		} else if !os.IsNotExist(e) {
			release()
			return nil, "git_boundary_unreadable"
		}
	}
	return handles, "ok"
}
func fixedFile(root, name string, read bool) ([]byte, string) {
	// There is no credential, log, environment or arbitrary relative-path option.
	switch name {
	case ".runner", ".service":
	case "bin/Runner.Listener.exe", "bin/RunnerService.exe", "config.cmd":
		if read {
			return nil, "not_allowed"
		}
	default:
		return nil, "not_allowed"
	}
	ptr, e := syscall.UTF16PtrFromString(filepath.Join(root, filepath.FromSlash(name)))
	if e != nil {
		return nil, "unsafe_path"
	}
	access := uint32(0x80)
	if read {
		access = syscall.GENERIC_READ
	}
	share := uint32(syscall.FILE_SHARE_READ | syscall.FILE_SHARE_WRITE)
	if read {
		share = syscall.FILE_SHARE_READ
	}
	h, e := syscall.CreateFile(ptr, access, share, nil, syscall.OPEN_EXISTING, syscall.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if e != nil {
		if errors.Is(e, syscall.ERROR_FILE_NOT_FOUND) || errors.Is(e, syscall.ERROR_PATH_NOT_FOUND) {
			return nil, "absent"
		}
		return nil, failure(e)
	}
	var info syscall.ByHandleFileInformation
	if syscall.GetFileInformationByHandle(h, &info) != nil || info.FileAttributes&(syscall.FILE_ATTRIBUTE_REPARSE_POINT|syscall.FILE_ATTRIBUTE_DIRECTORY) != 0 || (read && info.NumberOfLinks != 1) {
		syscall.CloseHandle(h)
		return nil, "unsafe_file"
	}
	if !read {
		syscall.CloseHandle(h)
		return nil, "present"
	}
	if info.FileSizeHigh != 0 || info.FileSizeLow > MaxConfigBytes {
		syscall.CloseHandle(h)
		return nil, "oversized"
	}
	f := os.NewFile(uintptr(h), "runner-metadata")
	defer f.Close()
	raw, e := io.ReadAll(io.LimitReader(f, MaxConfigBytes+1))
	if e != nil || len(raw) > MaxConfigBytes {
		return nil, "read_failed_or_oversized"
	}
	return raw, "readable"
}
func inspectRoot(c candidate) (Installation, string) {
	i := Installation{ID: opaqueID("runner", c.root), Source: c.source, WorkFolder: "not_observed", LocalState: "not_established"}
	locks, status := lockRoot(c.root)
	if status != "ok" {
		return i, status
	}
	defer func() {
		for _, h := range locks {
			syscall.CloseHandle(h)
		}
	}()
	// Pin the bin directory separately so a junction cannot redirect image checks.
	binLocks, binStatus := lockRoot(filepath.Join(c.root, "bin"))
	if binStatus == "ok" {
		_, i.Listener = fixedFile(c.root, "bin/Runner.Listener.exe", false)
		_, i.ServiceBinary = fixedFile(c.root, "bin/RunnerService.exe", false)
		for _, h := range binLocks {
			syscall.CloseHandle(h)
		}
	} else {
		i.Listener = binStatus
		i.ServiceBinary = binStatus
	}
	_, i.ConfigCommand = fixedFile(c.root, "config.cmd", false)
	_, i.ServiceMarker = fixedFile(c.root, ".service", false)
	raw, status := fixedFile(c.root, ".runner", true)
	i.Config = status
	if status == "readable" {
		settings, target, e := decodeSettings(raw)
		if e != nil {
			i.Config = "malformed_or_target_withheld"
		} else {
			i.AgentID = settings.AgentID
			i.Target = target
			i.WorkFolder = workFolderClass(settings.WorkFolder)
			i.Ephemeral = settings.Ephemeral
			i.DisableUpdate = settings.DisableUpdate
		}
	}
	if i.Listener == "present" && i.Config == "readable" {
		i.LocalState = "configured_locally_registration_unverified"
	} else if i.Listener == "present" && i.Config == "absent" {
		i.LocalState = "installed_without_runner_configuration"
	} else if i.Config == "readable" {
		i.LocalState = "configuration_present_installation_incomplete_or_unreadable"
	} else {
		i.LocalState = "candidate_only_not_a_proven_installation"
	}
	return i, "ok"
}
