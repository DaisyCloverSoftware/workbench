//go:build windows

package desktop

import (
	"errors"
	"strings"
	"syscall"
	"unsafe"
)

const (
	wsOverlappedWindow = 0x00CF0000
	wsVisible          = 0x10000000
	wsChild            = 0x40000000
	wsBorder           = 0x00800000
	wsVScroll          = 0x00200000
	wsTabStop          = 0x00010000
	wsExClientEdge     = 0x00000200

	esMultiline    = 0x0004
	esAutoVScroll  = 0x0040
	esAutoHScroll  = 0x0080
	esReadOnly     = 0x0800
	esPassword     = 0x0020
	lbsNotify      = 0x0001
	bsPushButton   = 0x00000000
	bsAutoCheckbox = 0x00000003

	wmCreate          = 0x0001
	wmDestroy         = 0x0002
	wmSize            = 0x0005
	wmCommand         = 0x0111
	wmClose           = 0x0010
	wmSetFont         = 0x0030
	wmCtlColorBtn     = 0x0135
	wmCtlColorEdit    = 0x0133
	wmCtlColorListBox = 0x0134
	wmCtlColorStatic  = 0x0138
	wmAppRefresh      = 0x8001

	lbAddString    = 0x0180
	lbResetContent = 0x0184
	lbGetCurSel    = 0x0188
	lbSetCurSel    = 0x0186
	lbnSelChange   = 1
	emSetCueBanner = 0x1501
	bmGetCheck     = 0x00F0
	bmSetCheck     = 0x00F1
	bstChecked     = 1

	swHide = 0
	swShow = 5

	mbOK              = 0x00000000
	mbIconError       = 0x00000010
	mbIconInformation = 0x00000040
	mbIconWarning     = 0x00000030
	mbYesNo           = 0x00000004
	idYes              = 6

	cfUnicodeText  = 13
	gmemMoveable   = 0x0002
	gmemZeroInit   = 0x0040
	defaultGUIFont = 17
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	dwmapi   = syscall.NewLazyDLL("dwmapi.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	ole32    = syscall.NewLazyDLL("ole32.dll")

	procRegisterClassExW      = user32.NewProc("RegisterClassExW")
	procCreateWindowExW       = user32.NewProc("CreateWindowExW")
	procDefWindowProcW        = user32.NewProc("DefWindowProcW")
	procShowWindow            = user32.NewProc("ShowWindow")
	procUpdateWindow          = user32.NewProc("UpdateWindow")
	procGetMessageW           = user32.NewProc("GetMessageW")
	procTranslateMessage      = user32.NewProc("TranslateMessage")
	procDispatchMessageW      = user32.NewProc("DispatchMessageW")
	procPostQuitMessage       = user32.NewProc("PostQuitMessage")
	procPostMessageW          = user32.NewProc("PostMessageW")
	procSendMessageW          = user32.NewProc("SendMessageW")
	procMoveWindow            = user32.NewProc("MoveWindow")
	procGetClientRect         = user32.NewProc("GetClientRect")
	procGetWindowTextLengthW  = user32.NewProc("GetWindowTextLengthW")
	procGetWindowTextW        = user32.NewProc("GetWindowTextW")
	procSetWindowTextW        = user32.NewProc("SetWindowTextW")
	procMessageBoxW           = user32.NewProc("MessageBoxW")
	procEnableWindow          = user32.NewProc("EnableWindow")
	procGetStockObject        = gdi32.NewProc("GetStockObject")
	procCreateSolidBrush      = gdi32.NewProc("CreateSolidBrush")
	procSetBkColor            = gdi32.NewProc("SetBkColor")
	procSetTextColor          = gdi32.NewProc("SetTextColor")
	procDwmSetWindowAttribute = dwmapi.NewProc("DwmSetWindowAttribute")
	procOpenClipboard         = user32.NewProc("OpenClipboard")
	procCloseClipboard        = user32.NewProc("CloseClipboard")
	procEmptyClipboard        = user32.NewProc("EmptyClipboard")
	procSetClipboardData      = user32.NewProc("SetClipboardData")
	procGlobalAlloc           = kernel32.NewProc("GlobalAlloc")
	procGlobalLock            = kernel32.NewProc("GlobalLock")
	procGlobalUnlock          = kernel32.NewProc("GlobalUnlock")
	procSHBrowseForFolderW    = shell32.NewProc("SHBrowseForFolderW")
	procSHGetPathFromIDListW  = shell32.NewProc("SHGetPathFromIDListW")
	procShellExecuteW         = shell32.NewProc("ShellExecuteW")
	procCoTaskMemFree         = ole32.NewProc("CoTaskMemFree")
)

type point struct{ X, Y int32 }
type nativeMsg struct {
	Hwnd           uintptr
	Message        uint32
	WParam, LParam uintptr
	Time           uint32
	Pt             point
	LPrivate       uint32
}
type nativeRect struct{ Left, Top, Right, Bottom int32 }
type wndClassEx struct {
	Size                                                            uint32
	Style                                                           uint32
	WndProc                                                         uintptr
	ClsExtra, WndExtra                                              int32
	Instance, Icon, Cursor, Background, MenuName, ClassName, IconSm uintptr
}
type browseInfo struct {
	Owner, Root        uintptr
	DisplayName, Title *uint16
	Flags              uint32
	Callback, LParam   uintptr
	Image              int32
}

func wstr(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}

func rgb(r, g, b byte) uint32 {
	return uint32(r) | uint32(g)<<8 | uint32(b)<<16
}

func defWindowProc(hwnd uintptr, message uint32, wParam, lParam uintptr) uintptr {
	if message == wmGetMinMaxInfo {
		enforceMinimumTrackSize(lParam)
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, uintptr(message), wParam, lParam)
	return r
}

func moveWindow(hwnd uintptr, x, y, width, height int) {
	if hwnd != 0 {
		procMoveWindow.Call(hwnd, uintptr(x), uintptr(y), uintptr(width), uintptr(height), 1)
	}
}

func showWindow(hwnd uintptr, visible bool) {
	if hwnd == 0 {
		return
	}
	cmd := uintptr(swHide)
	if visible {
		cmd = swShow
	}
	procShowWindow.Call(hwnd, cmd)
}

func setWindowText(hwnd uintptr, text string) {
	if hwnd == 0 {
		return
	}
	p := wstr(text)
	procSetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(p)))
}

func windowText(hwnd uintptr) string {
	if hwnd == 0 {
		return ""
	}
	length, _, _ := procGetWindowTextLengthW.Call(hwnd)
	buf := make([]uint16, int(length)+1)
	procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return syscall.UTF16ToString(buf)
}

func cueBanner(hwnd uintptr, text string) {
	if hwnd == 0 {
		return
	}
	p := wstr(text)
	procSendMessageW.Call(hwnd, emSetCueBanner, 1, uintptr(unsafe.Pointer(p)))
}

func listSelection(hwnd uintptr) int {
	r, _, _ := procSendMessageW.Call(hwnd, lbGetCurSel, 0, 0)
	if int32(r) < 0 {
		return -1
	}
	return int(r)
}

func setChecked(hwnd uintptr, checked bool) {
	value := uintptr(0)
	if checked {
		value = bstChecked
	}
	procSendMessageW.Call(hwnd, bmSetCheck, value, 0)
}

func isChecked(hwnd uintptr) bool {
	r, _, _ := procSendMessageW.Call(hwnd, bmGetCheck, 0, 0)
	return r == bstChecked
}

func messageBox(hwnd uintptr, title, body string, flags uintptr) int {
	t, b := wstr(title), wstr(body)
	r, _, _ := procMessageBoxW.Call(hwnd, uintptr(unsafe.Pointer(b)), uintptr(unsafe.Pointer(t)), flags)
	return int(r)
}

func copyText(hwnd uintptr, text string) error {
	if strings.TrimSpace(text) == "" {
		return errors.New("nothing to copy")
	}
	if r, _, e := procOpenClipboard.Call(hwnd); r == 0 {
		return e
	}
	defer procCloseClipboard.Call()
	procEmptyClipboard.Call()
	encoded, err := syscall.UTF16FromString(text)
	if err != nil {
		return err
	}
	size := uintptr(len(encoded) * 2)
	h, _, e := procGlobalAlloc.Call(gmemMoveable|gmemZeroInit, size)
	if h == 0 {
		return e
	}
	ptr, _, e := procGlobalLock.Call(h)
	if ptr == 0 {
		return e
	}
	dst := unsafe.Slice((*uint16)(unsafe.Pointer(ptr)), len(encoded))
	copy(dst, encoded)
	procGlobalUnlock.Call(h)
	if r, _, e := procSetClipboardData.Call(cfUnicodeText, h); r == 0 {
		return e
	}
	return nil
}

func browseFolder(owner uintptr, title string) string {
	display := make([]uint16, 260)
	t := wstr(title)
	bi := browseInfo{Owner: owner, DisplayName: &display[0], Title: t, Flags: 0x00000001 | 0x00000040}
	pidl, _, _ := procSHBrowseForFolderW.Call(uintptr(unsafe.Pointer(&bi)))
	if pidl == 0 {
		return ""
	}
	defer procCoTaskMemFree.Call(pidl)
	path := make([]uint16, 32768)
	ok, _, _ := procSHGetPathFromIDListW.Call(pidl, uintptr(unsafe.Pointer(&path[0])))
	if ok == 0 {
		return ""
	}
	return syscall.UTF16ToString(path)
}

func openExternal(owner uintptr, target string) error {
	target = strings.TrimSpace(target)
	if target == "" {
		return errors.New("target is empty")
	}
	verb := wstr("open")
	file := wstr(target)
	r, _, e := procShellExecuteW.Call(owner, uintptr(unsafe.Pointer(verb)), uintptr(unsafe.Pointer(file)), 0, 0, swShow)
	if r <= 32 {
		if e != nil && e != syscall.Errno(0) {
			return e
		}
		return errors.New("Windows could not open the target")
	}
	return nil
}
