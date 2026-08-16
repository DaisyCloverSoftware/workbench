//go:build windows

package desktop

import (
	"syscall"
	"unsafe"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

const (
	runnerPickerListID     = 3911
	runnerPickerAddID      = 3912
	runnerPickerAddAllID   = 3913
	runnerPickerCancelID   = 3914
	runnerPickerWidth      = 620
	runnerPickerHeight     = 460
	runnerPickerListDblClk = 2
)

type runnerProjectPickerState struct {
	hwnd     uintptr
	parent   uintptr
	list     uintptr
	projects []core.RunnerProjectInfo
	selected []core.RunnerProjectInfo
	done     bool
}

var activeRunnerProjectPicker *runnerProjectPickerState
var runnerProjectPickerClassRegistered bool
var runnerProjectPickerBrush uintptr

func chooseRunnerProjects(parent uintptr, projects []core.RunnerProjectInfo) ([]core.RunnerProjectInfo, bool) {
	if len(projects) == 0 {
		return nil, false
	}
	if !registerRunnerProjectPickerClass() {
		return nil, false
	}
	state := &runnerProjectPickerState{
		parent:   parent,
		projects: append([]core.RunnerProjectInfo(nil), projects...),
	}
	activeRunnerProjectPicker = state
	defer func() { activeRunnerProjectPicker = nil }()

	instance, _, _ := kernel32.NewProc("GetModuleHandleW").Call(0)
	className := wstr("DaisyCloverWorkbenchRunnerProjectPicker")
	title := wstr("Choose cluster project")
	// Caption + system menu only: this is a small modal chooser, not another
	// independent Workbench window.
	style := uintptr(0x00C80000)
	hwnd, _, _ := procCreateWindowExW.Call(
		0x00000001, // WS_EX_DLGMODALFRAME
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(title)),
		style,
		160, 120, runnerPickerWidth, runnerPickerHeight,
		parent, 0, instance, 0,
	)
	if hwnd == 0 {
		return nil, false
	}
	state.hwnd = hwnd
	procEnableWindow.Call(parent, 0)
	defer func() {
		procEnableWindow.Call(parent, 1)
		focusWindow(parent)
	}()
	procShowWindow.Call(hwnd, swShow)
	procUpdateWindow.Call(hwnd)

	var message nativeMsg
	for !state.done {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&message)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&message)))
	}
	if len(state.selected) == 0 {
		return nil, false
	}
	return append([]core.RunnerProjectInfo(nil), state.selected...), true
}

func registerRunnerProjectPickerClass() bool {
	if runnerProjectPickerClassRegistered {
		return true
	}
	instance, _, _ := kernel32.NewProc("GetModuleHandleW").Call(0)
	if runnerProjectPickerBrush == 0 {
		runnerProjectPickerBrush, _, _ = procCreateSolidBrush.Call(uintptr(productionPalette.Panel))
	}
	className := wstr("DaisyCloverWorkbenchRunnerProjectPicker")
	cursor, _, _ := user32.NewProc("LoadCursorW").Call(0, 32512)
	wc := wndClassEx{
		Size:       uint32(unsafe.Sizeof(wndClassEx{})),
		WndProc:    syscall.NewCallback(runnerProjectPickerWndProc),
		Instance:   instance,
		Cursor:     cursor,
		Background: runnerProjectPickerBrush,
		ClassName:  uintptr(unsafe.Pointer(className)),
	}
	r, _, _ := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	if r == 0 {
		return false
	}
	runnerProjectPickerClassRegistered = true
	return true
}

func runnerProjectPickerWndProc(hwnd uintptr, message uint32, wParam, lParam uintptr) uintptr {
	state := activeRunnerProjectPicker
	if state == nil {
		return defWindowProc(hwnd, message, wParam, lParam)
	}
	switch message {
	case wmCreate:
		state.hwnd = hwnd
		createRunnerProjectPickerControls(state)
		return 0
	case wmCommand:
		id := int(uint16(wParam & 0xffff))
		notify := uint16((wParam >> 16) & 0xffff)
		switch id {
		case runnerPickerListID:
			if notify == runnerPickerListDblClk {
				selectRunnerPickerProject(state)
			}
		case runnerPickerAddID:
			selectRunnerPickerProject(state)
		case runnerPickerAddAllID:
			state.selected = append([]core.RunnerProjectInfo(nil), state.projects...)
			user32.NewProc("DestroyWindow").Call(hwnd)
		case runnerPickerCancelID:
			user32.NewProc("DestroyWindow").Call(hwnd)
		}
		return 0
	case wmCtlColorStatic:
		hdc := wParam
		procSetTextColor.Call(hdc, uintptr(productionPalette.Text))
		procSetBkColor.Call(hdc, uintptr(productionPalette.Panel))
		return runnerProjectPickerBrush
	case wmClose:
		user32.NewProc("DestroyWindow").Call(hwnd)
		return 0
	case wmDestroy:
		state.done = true
		return 0
	}
	return defWindowProc(hwnd, message, wParam, lParam)
}

func createRunnerProjectPickerControls(state *runnerProjectPickerState) {
	instance, _, _ := kernel32.NewProc("GetModuleHandleW").Call(0)
	font, _, _ := procGetStockObject.Call(defaultGUIFont)
	create := func(id int, class, text string, style uintptr, x, y, width, height int) uintptr {
		classPtr, textPtr := wstr(class), wstr(text)
		hwnd, _, _ := procCreateWindowExW.Call(
			0,
			uintptr(unsafe.Pointer(classPtr)), uintptr(unsafe.Pointer(textPtr)), style,
			uintptr(x), uintptr(y), uintptr(width), uintptr(height),
			state.hwnd, uintptr(id), instance, 0,
		)
		procSendMessageW.Call(hwnd, wmSetFont, font, 1)
		return hwnd
	}
	create(0, "STATIC", "Choose one repository to add, or add all discovered repositories. They remain on the cluster runner.", wsChild|wsVisible, 20, 18, 570, 38)
	state.list = create(runnerPickerListID, "LISTBOX", "", wsChild|wsVisible|wsBorder|wsVScroll|lbsNotify, 20, 62, 570, 300)
	applyDarkExplorerTheme(state.list)
	for _, project := range state.projects {
		line := project.Name + "   ·   " + project.Ref
		ptr := wstr(line)
		procSendMessageW.Call(state.list, lbAddString, 0, uintptr(unsafe.Pointer(ptr)))
	}
	if len(state.projects) > 0 {
		procSendMessageW.Call(state.list, lbSetCurSel, 0, 0)
	}
	create(runnerPickerAddID, "BUTTON", "Add selected", wsChild|wsVisible|wsTabStop|bsPushButton, 20, 378, 130, 34)
	create(runnerPickerAddAllID, "BUTTON", "Add all", wsChild|wsVisible|wsTabStop|bsPushButton, 160, 378, 110, 34)
	create(runnerPickerCancelID, "BUTTON", "Cancel", wsChild|wsVisible|wsTabStop|bsPushButton, 480, 378, 110, 34)
}

func selectRunnerPickerProject(state *runnerProjectPickerState) {
	idx := listSelection(state.list)
	if idx < 0 || idx >= len(state.projects) {
		messageBox(state.hwnd, "Choose cluster project", "Select a repository first.", mbOK|mbIconInformation)
		return
	}
	state.selected = []core.RunnerProjectInfo{state.projects[idx]}
	user32.NewProc("DestroyWindow").Call(state.hwnd)
}

func runnerProjectSelection(projects []core.RunnerProjectInfo, index int, all bool) []core.RunnerProjectInfo {
	if all {
		return append([]core.RunnerProjectInfo(nil), projects...)
	}
	if index < 0 || index >= len(projects) {
		return nil
	}
	return []core.RunnerProjectInfo{projects[index]}
}
