//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

const (
	WS_OVERLAPPEDWINDOW = 0x00CF0000
	WS_VISIBLE          = 0x10000000
	WS_CHILD            = 0x40000000
	WS_BORDER           = 0x00800000
	WS_VSCROLL          = 0x00200000
	WS_TABSTOP          = 0x00010000
	WS_EX_CLIENTEDGE    = 0x00000200
	SW_SHOW             = 5

	ES_MULTILINE   = 0x0004
	ES_AUTOVSCROLL = 0x0040
	ES_AUTOHSCROLL = 0x0080
	ES_READONLY    = 0x0800
	LBS_NOTIFY     = 0x0001
	BS_PUSHBUTTON  = 0x00000000

	WM_CREATE          = 0x0001
	WM_DESTROY         = 0x0002
	WM_SIZE            = 0x0005
	WM_COMMAND         = 0x0111
	WM_CLOSE           = 0x0010
	WM_SETFONT         = 0x0030
	WM_CTLCOLORBTN     = 0x0135
	WM_CTLCOLOREDIT    = 0x0133
	WM_CTLCOLORLISTBOX = 0x0134
	WM_CTLCOLORSTATIC  = 0x0138
	WM_APP_REFRESH     = 0x8001

	LB_ADDSTRING    = 0x0180
	LB_RESETCONTENT = 0x0184
	LB_GETCURSEL    = 0x0188
	LB_SETCURSEL    = 0x0186
	LBN_SELCHANGE   = 1
	EM_SETCUEBANNER = 0x1501

	MB_OK              = 0x00000000
	MB_ICONERROR       = 0x00000010
	MB_ICONINFORMATION = 0x00000040
	MB_ICONWARNING     = 0x00000030

	CF_UNICODETEXT = 13
	GMEM_MOVEABLE  = 0x0002
	GMEM_ZEROINIT  = 0x0040
	DEFAULT_GUI_FONT = 17
)

const (
	idTitle          = 2001
	idSummary        = 2002
	idProjectLabel   = 2003
	idProject        = 2004
	idBrowse         = 2005
	idIntentLabel    = 2006
	idIntent         = 2007
	idStart          = 2008
	idCancel         = 2009
	idTasksLabel     = 2010
	idTaskList       = 2011
	idReportLabel    = 2012
	idReport         = 2013
	idAttentionLabel = 2014
	idAnswer         = 2015
	idResume         = 2016
	idCopyReview     = 2017
	idProviderStatus = 2018
	idRescan         = 2019
	idAdvanced       = 2020
	idUpdater        = 2021
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
	procCoTaskMemFree         = ole32.NewProc("CoTaskMemFree")
)

type point struct{ X, Y int32 }
type msg struct {
	Hwnd           uintptr
	Message        uint32
	WParam, LParam uintptr
	Time           uint32
	Pt             point
	LPrivate       uint32
}
type rect struct{ Left, Top, Right, Bottom int32 }
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

type dashboard struct {
	hwnd               uintptr
	eng                *core.Engine
	font               uintptr
	brush              uintptr
	controls           map[int]uintptr
	taskIDs            []string
	focusTaskID        string
	projectInitialized bool
}

var app *dashboard

func main() {
	store, err := core.NewStore()
	if err != nil {
		fatal("Workbench could not open its state", err)
		return
	}
	eng, err := core.NewEngine(store)
	if err != nil {
		fatal("Workbench could not initialise", err)
		return
	}
	app = &dashboard{eng: eng, controls: map[int]uintptr{}}
	if err := app.run(); err != nil {
		fatal("Workbench Dogfood failed", err)
	}
}

func (a *dashboard) run() error {
	instance, _, _ := kernel32.NewProc("GetModuleHandleW").Call(0)
	className := wstr("WorkbenchDogfoodWindow")
	icon, _, _ := user32.NewProc("LoadIconW").Call(0, 32512)
	cursor, _, _ := user32.NewProc("LoadCursorW").Call(0, 32512)
	a.brush, _, _ = procCreateSolidBrush.Call(uintptr(rgb(22, 25, 30)))
	wc := wndClassEx{Size: uint32(unsafe.Sizeof(wndClassEx{})), WndProc: syscall.NewCallback(wndProc), Instance: instance, Icon: icon, Cursor: cursor, Background: a.brush, ClassName: uintptr(unsafe.Pointer(className)), IconSm: icon}
	if r, _, e := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		return e
	}
	title := wstr("Workbench Dogfood " + core.Version + " — Get work done")
	hwnd, _, e := procCreateWindowExW.Call(0, uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(title)), WS_OVERLAPPEDWINDOW|WS_VISIBLE, 100, 70, 1380, 860, 0, 0, instance, 0)
	if hwnd == 0 {
		return e
	}
	a.hwnd = hwnd
	useDark := uint32(1)
	_, _, _ = procDwmSetWindowAttribute.Call(hwnd, 20, uintptr(unsafe.Pointer(&useDark)), unsafe.Sizeof(useDark))
	a.eng.Subscribe(func() {
		if a.hwnd != 0 {
			procPostMessageW.Call(a.hwnd, WM_APP_REFRESH, 0, 0)
		}
	})
	procShowWindow.Call(hwnd, SW_SHOW)
	procUpdateWindow.Call(hwnd)
	var m msg
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
	return nil
}

func wndProc(hwnd uintptr, message uint32, wParam, lParam uintptr) uintptr {
	if app == nil {
		return def(hwnd, message, wParam, lParam)
	}
	switch message {
	case WM_CREATE:
		app.hwnd = hwnd
		app.createControls()
		app.refreshAll()
		app.layout()
		return 0
	case WM_SIZE:
		app.layout()
		return 0
	case WM_APP_REFRESH:
		app.refreshAll()
		return 0
	case WM_COMMAND:
		id := int(uint16(wParam & 0xffff))
		notify := uint16((wParam >> 16) & 0xffff)
		app.command(id, notify)
		return 0
	case WM_CTLCOLORSTATIC, WM_CTLCOLORBTN, WM_CTLCOLOREDIT, WM_CTLCOLORLISTBOX:
		hdc := wParam
		procSetTextColor.Call(hdc, uintptr(rgb(236, 239, 244)))
		procSetBkColor.Call(hdc, uintptr(rgb(22, 25, 30)))
		return app.brush
	case WM_CLOSE:
		user32.NewProc("DestroyWindow").Call(hwnd)
		return 0
	case WM_DESTROY:
		procPostQuitMessage.Call(0)
		return 0
	}
	return def(hwnd, message, wParam, lParam)
}

func def(hwnd uintptr, message uint32, wParam, lParam uintptr) uintptr {
	r, _, _ := procDefWindowProcW.Call(hwnd, uintptr(message), wParam, lParam)
	return r
}

func (a *dashboard) createControls() {
	a.font, _, _ = procGetStockObject.Call(DEFAULT_GUI_FONT)
	a.static("WORKBENCH  ·  TELL IT WHAT YOU WANT, THEN GO DO SOMETHING ELSE", idTitle)
	a.static("", idSummary)
	a.static("Project", idProjectLabel)
	a.ctrl(idProject, "EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|ES_AUTOHSCROLL)
	a.ctrl(idBrowse, "BUTTON", "Browse…", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON)
	a.static("What should Workbench get done?", idIntentLabel)
	a.ctrl(idIntent, "EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|ES_MULTILINE|ES_AUTOVSCROLL|WS_VSCROLL)
	a.ctrl(idStart, "BUTTON", "Start task", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON)
	a.ctrl(idCancel, "BUTTON", "Cancel selected", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON)
	a.static("Work", idTasksLabel)
	a.ctrl(idTaskList, "LISTBOX", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_VSCROLL|LBS_NOTIFY)
	a.static("Selected task", idReportLabel)
	a.ctrl(idReport, "EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|ES_MULTILINE|ES_AUTOVSCROLL|WS_VSCROLL|ES_READONLY)
	a.static("", idAttentionLabel)
	a.ctrl(idAnswer, "EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_TABSTOP|ES_AUTOHSCROLL)
	a.ctrl(idResume, "BUTTON", "Answer + continue", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON)
	a.ctrl(idCopyReview, "BUTTON", "Copy review ref", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON)
	a.static("", idProviderStatus)
	a.ctrl(idRescan, "BUTTON", "Rescan workers", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON)
	a.ctrl(idAdvanced, "BUTTON", "Advanced settings", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON)
	a.ctrl(idUpdater, "BUTTON", "Update Workbench", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON)

	cue(a.controls[idProject], "Choose the repository Workbench may edit")
	cue(a.controls[idIntent], "e.g. Implement the next Workbench task. Fix it, test it and leave the result ready for review.")
	cue(a.controls[idAnswer], "Only answer here when the selected task says NEEDS YOU")
}

func (a *dashboard) static(text string, id int) uintptr {
	return a.ctrl(id, "STATIC", text, WS_CHILD|WS_VISIBLE)
}

func (a *dashboard) ctrl(id int, class, text string, style uintptr) uintptr {
	instance, _, _ := kernel32.NewProc("GetModuleHandleW").Call(0)
	c, t := wstr(class), wstr(text)
	exStyle := uintptr(0)
	if class == "EDIT" || class == "LISTBOX" {
		exStyle = WS_EX_CLIENTEDGE
	}
	h, _, _ := procCreateWindowExW.Call(exStyle, uintptr(unsafe.Pointer(c)), uintptr(unsafe.Pointer(t)), style, 0, 0, 10, 10, a.hwnd, uintptr(id), instance, 0)
	a.controls[id] = h
	procSendMessageW.Call(h, WM_SETFONT, a.font, 1)
	return h
}

func (a *dashboard) layout() {
	if a.hwnd == 0 {
		return
	}
	var r rect
	procGetClientRect.Call(a.hwnd, uintptr(unsafe.Pointer(&r)))
	w := int(r.Right - r.Left)
	h := int(r.Bottom - r.Top)
	pad := 18
	contentW := w - pad*2
	left := 390
	if w < 1120 {
		left = 330
	}
	rightX := pad + left + pad
	rightW := w - rightX - pad

	move(a.controls[idTitle], pad, 12, contentW, 22)
	move(a.controls[idSummary], pad, 38, contentW, 24)
	move(a.controls[idProjectLabel], pad, 72, 100, 18)
	move(a.controls[idProject], pad, 92, contentW-110, 30)
	move(a.controls[idBrowse], w-pad-100, 92, 100, 30)
	move(a.controls[idIntentLabel], pad, 132, contentW, 20)
	move(a.controls[idIntent], pad, 154, contentW, 88)
	move(a.controls[idStart], pad, 250, 170, 36)
	move(a.controls[idCancel], pad+180, 250, 140, 36)
	move(a.controls[idTasksLabel], pad, 300, left, 20)
	move(a.controls[idTaskList], pad, 322, left, h-410)
	move(a.controls[idProviderStatus], pad, h-80, left, 52)
	move(a.controls[idReportLabel], rightX, 300, rightW, 20)
	move(a.controls[idReport], rightX, 322, rightW, h-502)
	move(a.controls[idAttentionLabel], rightX, h-172, rightW, 22)
	move(a.controls[idAnswer], rightX, h-145, rightW-170, 31)
	move(a.controls[idResume], rightX+rightW-160, h-145, 160, 31)
	move(a.controls[idCopyReview], rightX, h-104, 140, 32)
	move(a.controls[idRescan], rightX+150, h-104, 130, 32)
	move(a.controls[idAdvanced], rightX+290, h-104, 140, 32)
	move(a.controls[idUpdater], rightX+440, h-104, 140, 32)
}

func move(hwnd uintptr, x, y, w, h int) {
	if hwnd != 0 && w > 0 && h > 0 {
		procMoveWindow.Call(hwnd, uintptr(x), uintptr(y), uintptr(w), uintptr(h), 1)
	}
}

func (a *dashboard) refreshAll() {
	st := a.eng.State()
	if !a.projectInitialized {
		setText(a.controls[idProject], st.ProjectPath)
		a.projectInitialized = true
	}
	summary := core.SummarizeTasks(st.Tasks)
	setText(a.controls[idSummary], fmt.Sprintf("%d working  ·  %d need you  ·  %d ready  ·  %d failed", summary.Active, summary.NeedsHuman, summary.Completed, summary.Failed))
	a.refreshTasks(st.Tasks)
	a.refreshProviders()
}

func (a *dashboard) refreshTasks(tasks []core.Task) {
	selectedID := a.selectedTaskID()
	if a.focusTaskID != "" {
		selectedID = a.focusTaskID
	}
	procSendMessageW.Call(a.controls[idTaskList], LB_RESETCONTENT, 0, 0)
	a.taskIDs = nil
	selected := -1
	for i, task := range tasks {
		p := core.PresentTask(task)
		line := fmt.Sprintf("%s  %s  ·  %s", taskMarker(task.Status), task.Title, p.ProviderLabel)
		ptr := wstr(line)
		procSendMessageW.Call(a.controls[idTaskList], LB_ADDSTRING, 0, uintptr(unsafe.Pointer(ptr)))
		a.taskIDs = append(a.taskIDs, task.ID)
		if task.ID == selectedID {
			selected = i
		}
	}
	if len(tasks) == 0 {
		setText(a.controls[idReport], "No work yet.\r\n\r\nChoose a project, describe the outcome you want and click Start task. Workbench will route it, keep going without supervision and only interrupt you for a genuine decision.")
		setText(a.controls[idAttentionLabel], "Nothing needs you.")
		procEnableWindow.Call(a.controls[idAnswer], 0)
		procEnableWindow.Call(a.controls[idResume], 0)
		procEnableWindow.Call(a.controls[idCopyReview], 0)
		return
	}
	if selected < 0 {
		selected = 0
	}
	procSendMessageW.Call(a.controls[idTaskList], LB_SETCURSEL, uintptr(selected), 0)
	a.focusTaskID = ""
	a.showTask(selected)
}

func taskMarker(status core.TaskStatus) string {
	switch status {
	case core.TaskQueued, core.TaskRouting:
		return "○"
	case core.TaskRunning:
		return "▶"
	case core.TaskNeedsAttention:
		return "!"
	case core.TaskCompleted:
		return "✓"
	case core.TaskFailed:
		return "×"
	case core.TaskCancelled:
		return "–"
	default:
		return "·"
	}
}

func (a *dashboard) showTask(index int) {
	st := a.eng.State()
	if index < 0 || index >= len(st.Tasks) {
		return
	}
	task := st.Tasks[index]
	p := core.PresentTask(task)
	var b strings.Builder
	fmt.Fprintf(&b, "%s\r\n\r\n%s\r\n\r\nNEXT\r\n%s\r\n\r\n", task.Title, strings.ToUpper(p.StatusLabel), p.NextAction)
	fmt.Fprintf(&b, "Worker: %s\r\n", p.ProviderLabel)
	if task.StartedAt != nil {
		fmt.Fprintf(&b, "Started: %s\r\n", task.StartedAt.Local().Format("02 Jan 15:04"))
	}
	if task.FinishedAt != nil {
		fmt.Fprintf(&b, "Finished: %s\r\n", task.FinishedAt.Local().Format("02 Jan 15:04"))
	}
	if p.ReviewBranch != "" {
		fmt.Fprintf(&b, "Review branch: %s\r\nReview commit: %s\r\nPublished: %t\r\n", p.ReviewBranch, p.ReviewCommit, p.Published)
	}
	b.WriteString("\r\nGOAL\r\n")
	b.WriteString(strings.TrimSpace(task.Intent))
	b.WriteString("\r\n\r\n")
	if task.AttentionQuestion != "" {
		b.WriteString("NEEDS YOU\r\n")
		b.WriteString(task.AttentionQuestion)
		b.WriteString("\r\n\r\n")
	}
	if strings.TrimSpace(task.Output) != "" {
		b.WriteString("RESULT\r\n")
		b.WriteString(strings.TrimSpace(task.Output))
		b.WriteString("\r\n\r\n")
	}
	if strings.TrimSpace(task.Error) != "" {
		b.WriteString("WHY IT STOPPED\r\n")
		b.WriteString(strings.TrimSpace(task.Error))
		b.WriteString("\r\n\r\n")
	}
	if len(task.Attempts) > 0 && (task.Status == core.TaskFailed || task.Status == core.TaskNeedsAttention) {
		b.WriteString("ROUTING DETAILS\r\n- ")
		b.WriteString(strings.Join(task.Attempts, "\r\n- "))
	}
	setText(a.controls[idReport], b.String())
	if p.NeedsHuman {
		setText(a.controls[idAttentionLabel], "NEEDS YOU — answer this one decision and Workbench will continue")
	} else {
		setText(a.controls[idAttentionLabel], "Nothing needs you for this task.")
	}
	procEnableWindow.Call(a.controls[idAnswer], boolPtr(p.NeedsHuman))
	procEnableWindow.Call(a.controls[idResume], boolPtr(p.NeedsHuman))
	procEnableWindow.Call(a.controls[idCopyReview], boolPtr(p.ReviewBranch != ""))
}

func (a *dashboard) refreshProviders() {
	providers := a.eng.Providers()
	var ready []string
	for _, p := range providers {
		if p.Installed && p.Authenticated && p.CanWrite {
			ready = append(ready, p.Name)
		}
	}
	status := "No coding workers are ready. Open Advanced settings to connect one."
	if len(ready) > 0 {
		shown := ready
		if len(shown) > 3 {
			shown = shown[:3]
		}
		status = fmt.Sprintf("Workers ready: %s", strings.Join(shown, " · "))
		if len(ready) > len(shown) {
			status += fmt.Sprintf(" · +%d more", len(ready)-len(shown))
		}
		status += "\r\nWorkbench chooses automatically; scarce Work stays protected by your saved policy."
	}
	setText(a.controls[idProviderStatus], status)
}

func (a *dashboard) command(id int, notify uint16) {
	switch id {
	case idTaskList:
		if notify == LBN_SELCHANGE {
			a.showTask(listSel(a.controls[idTaskList]))
		}
	case idBrowse:
		if path := browseFolder(a.hwnd); path != "" {
			setText(a.controls[idProject], path)
		}
	case idStart:
		a.startTask()
	case idCancel:
		if taskID := a.selectedTaskID(); taskID != "" {
			if err := a.eng.Cancel(taskID); err != nil {
				msgbox(a.hwnd, "Cannot cancel task", err.Error(), MB_ICONWARNING)
			}
		}
	case idResume:
		taskID := a.selectedTaskID()
		answer := strings.TrimSpace(getText(a.controls[idAnswer]))
		if err := a.eng.ResolveAttention(taskID, answer); err != nil {
			msgbox(a.hwnd, "Cannot continue", err.Error(), MB_ICONWARNING)
			return
		}
		setText(a.controls[idAnswer], "")
	case idCopyReview:
		a.copyReview()
	case idRescan:
		go a.eng.RescanProviders()
	case idAdvanced:
		a.launchSibling("Workbench.exe", "Advanced Workbench")
	case idUpdater:
		a.launchSibling("Workbench-Updater.exe", "Workbench Updater")
	}
}

func (a *dashboard) startTask() {
	intent := strings.TrimSpace(getText(a.controls[idIntent]))
	project := strings.TrimSpace(getText(a.controls[idProject]))
	task, err := a.eng.Delegate("dogfood-desktop", intent, project)
	if err != nil {
		msgbox(a.hwnd, "Cannot start task", err.Error(), MB_ICONWARNING)
		return
	}
	a.focusTaskID = task.ID
	setText(a.controls[idIntent], "")
}

func (a *dashboard) copyReview() {
	idx := listSel(a.controls[idTaskList])
	st := a.eng.State()
	if idx < 0 || idx >= len(st.Tasks) {
		return
	}
	p := core.PresentTask(st.Tasks[idx])
	if p.ReviewBranch == "" {
		msgbox(a.hwnd, "Review", "The selected task does not have a prepared review branch yet.", MB_ICONINFORMATION)
		return
	}
	text := p.ReviewBranch + " @ " + p.ReviewCommit
	if p.Published {
		text += " (published)"
	}
	if copyClipboard(a.hwnd, text) {
		msgbox(a.hwnd, "Copied", "Review branch and commit copied to the clipboard.", MB_ICONINFORMATION)
	}
}

func (a *dashboard) launchSibling(name, label string) {
	exe, err := os.Executable()
	if err != nil {
		msgbox(a.hwnd, label, err.Error(), MB_ICONERROR)
		return
	}
	target := filepath.Join(filepath.Dir(exe), name)
	if st, err := os.Stat(target); err != nil || st.IsDir() {
		msgbox(a.hwnd, label, name+" was not found beside Workbench-Dogfood.exe.", MB_ICONINFORMATION)
		return
	}
	if err := exec.Command(target).Start(); err != nil {
		msgbox(a.hwnd, label, err.Error(), MB_ICONERROR)
		return
	}
	user32.NewProc("DestroyWindow").Call(a.hwnd)
}

func (a *dashboard) selectedTaskID() string {
	idx := listSel(a.controls[idTaskList])
	if idx >= 0 && idx < len(a.taskIDs) {
		return a.taskIDs[idx]
	}
	return ""
}

func wstr(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}

func getText(hwnd uintptr) string {
	if hwnd == 0 {
		return ""
	}
	n, _, _ := procGetWindowTextLengthW.Call(hwnd)
	buf := make([]uint16, n+1)
	procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), n+1)
	return syscall.UTF16ToString(buf)
}

func setText(hwnd uintptr, text string) {
	if hwnd == 0 {
		return
	}
	p := wstr(text)
	procSetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(p)))
}

func cue(hwnd uintptr, text string) {
	if hwnd == 0 {
		return
	}
	p := wstr(text)
	procSendMessageW.Call(hwnd, EM_SETCUEBANNER, 1, uintptr(unsafe.Pointer(p)))
}

func listSel(hwnd uintptr) int {
	r, _, _ := procSendMessageW.Call(hwnd, LB_GETCURSEL, 0, 0)
	if int32(r) < 0 {
		return -1
	}
	return int(r)
}

func boolPtr(v bool) uintptr {
	if v {
		return 1
	}
	return 0
}

func rgb(r, g, b byte) uint32 { return uint32(r) | uint32(g)<<8 | uint32(b)<<16 }

func msgbox(hwnd uintptr, title, text string, flags uintptr) int {
	t, m := wstr(title), wstr(text)
	r, _, _ := procMessageBoxW.Call(hwnd, uintptr(unsafe.Pointer(m)), uintptr(unsafe.Pointer(t)), flags|MB_OK)
	return int(r)
}

func fatal(title string, err error) { msgbox(0, title, err.Error(), MB_ICONERROR) }

func copyClipboard(hwnd uintptr, text string) bool {
	r, _, _ := procOpenClipboard.Call(hwnd)
	if r == 0 {
		return false
	}
	defer procCloseClipboard.Call()
	procEmptyClipboard.Call()
	u, _ := syscall.UTF16FromString(text)
	size := uintptr(len(u) * 2)
	h, _, _ := procGlobalAlloc.Call(GMEM_MOVEABLE|GMEM_ZEROINIT, size)
	if h == 0 {
		return false
	}
	p, _, _ := procGlobalLock.Call(h)
	if p == 0 {
		return false
	}
	dst := unsafe.Slice((*uint16)(unsafe.Pointer(p)), len(u))
	copy(dst, u)
	procGlobalUnlock.Call(h)
	r, _, _ = procSetClipboardData.Call(CF_UNICODETEXT, h)
	return r != 0
}

func browseFolder(owner uintptr) string {
	display := make([]uint16, 260)
	title := wstr("Choose the project/repository Workbench may edit")
	bi := browseInfo{Owner: owner, DisplayName: &display[0], Title: title, Flags: 0x0001 | 0x0040}
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

var _ = time.Now
