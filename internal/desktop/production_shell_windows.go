//go:build windows

package desktop

import (
	"context"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

func runProductionShell(s *Shell) error {
	instance, _, _ := kernel32.NewProc("GetModuleHandleW").Call(0)
	className := wstr("DaisyCloverWorkbenchProductionDashboard")
	icon, _, _ := user32.NewProc("LoadIconW").Call(0, 32512)
	cursor, _, _ := user32.NewProc("LoadCursorW").Call(0, 32512)
	s.backgroundBrush, _, _ = procCreateSolidBrush.Call(uintptr(productionPalette.Background))
	wc := wndClassEx{
		Size:       uint32(unsafe.Sizeof(wndClassEx{})),
		WndProc:    syscall.NewCallback(productionShellWndProc),
		Instance:   instance,
		Icon:       icon,
		Cursor:     cursor,
		Background: s.backgroundBrush,
		ClassName:  uintptr(unsafe.Pointer(className)),
		IconSm:     icon,
	}
	if r, _, e := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		return e
	}
	title := "Workbench"
	if strings.TrimSpace(s.version) != "" {
		title += " " + strings.TrimSpace(s.version)
	}
	title += " — Autonomous developer workspace"
	titlePtr := wstr(title)
	hwnd, _, e := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(titlePtr)),
		wsOverlappedWindow|wsVisible,
		48, 36, 1560, 960,
		0, 0, instance, 0,
	)
	if hwnd == 0 {
		return e
	}
	s.hwnd = hwnd
	useDark := uint32(1)
	_, _, _ = procDwmSetWindowAttribute.Call(hwnd, 20, uintptr(unsafe.Pointer(&useDark)), unsafe.Sizeof(useDark))
	s.eng.Subscribe(func() {
		if s.hwnd != 0 {
			procPostMessageW.Call(s.hwnd, wmAppRefresh, 0, 0)
		}
	})
	procShowWindow.Call(hwnd, swShow)
	procUpdateWindow.Call(hwnd)

	var message nativeMsg
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&message)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&message)))
	}
	if s.mcp != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		_ = s.mcp.Close(ctx)
		cancel()
	}
	if s.backgroundBrush != 0 {
		procDeleteObject.Call(s.backgroundBrush)
		s.backgroundBrush = 0
	}
	return nil
}

func productionShellWndProc(hwnd uintptr, message uint32, wParam, lParam uintptr) uintptr {
	s := runningShell
	if s == nil {
		return defWindowProc(hwnd, message, wParam, lParam)
	}
	switch message {
	case wmCreate:
		s.hwnd = hwnd
		s.createControls()
		s.createDashboardChrome()
		s.styleProductionControls()
		brand := "Workbench"
		if strings.TrimSpace(s.version) != "" {
			brand += "  " + strings.TrimSpace(s.version)
		}
		setWindowText(s.controls[idBrand], brand)
		showWindow(s.controls[idGlobalStatus], false)
		s.refresh()
		s.applyPageVisibility()
		s.layoutProduction()
		redrawProductionWindow(hwnd)
		return 0
	case wmSize:
		s.layoutProduction()
		redrawProductionWindow(hwnd)
		return 0
	case wmPaint:
		result := s.paintProductionWindow()
		if s.page != pageDashboard {
			s.paintProductionWorkSettingsPanels()
		}
		return result
	case wmEraseBkgnd:
		return 1
	case wmDrawItem:
		if result := s.drawProductionButton(lParam); result != 0 {
			return result
		}
		if result := s.drawGenericProductionButton(lParam); result != 0 {
			return result
		}
	case wmAppRefresh:
		s.refresh()
		showWindow(s.controls[idGlobalStatus], false)
		redrawProductionWindow(hwnd)
		return 0
	case wmCommand:
		id := int(uint16(wParam & 0xffff))
		notify := uint16((wParam >> 16) & 0xffff)
		if s.handleProductionChromeCommand(id) {
			redrawProductionWindow(hwnd)
			return 0
		}
		s.handleCommand(id, notify)
		showWindow(s.controls[idGlobalStatus], false)
		redrawProductionWindow(hwnd)
		return 0
	case wmCtlColorStatic, wmCtlColorBtn, wmCtlColorEdit, wmCtlColorListBox:
		hdc := wParam
		procSetTextColor.Call(hdc, uintptr(productionPalette.Text))
		procSetBkColor.Call(hdc, uintptr(productionPalette.Background))
		return s.backgroundBrush
	case wmClose:
		user32.NewProc("DestroyWindow").Call(hwnd)
		return 0
	case wmDestroy:
		procPostQuitMessage.Call(0)
		return 0
	}
	return defWindowProc(hwnd, message, wParam, lParam)
}

func (s *Shell) layoutProduction() {
	if s.hwnd == 0 {
		return
	}
	var r nativeRect
	procGetClientRect.Call(s.hwnd, uintptr(unsafe.Pointer(&r)))
	width := int(r.Right - r.Left)
	height := int(r.Bottom - r.Top)
	if width < 1360 {
		width = 1360
	}
	pad := 16
	contentX := productionSidebarWidth + pad
	contentW := width - contentX - pad
	contentH := height - productionHeaderHeight - pad

	s.layoutProductionChrome(width)
	showWindow(s.controls[idGlobalStatus], false)
	switch s.page {
	case pageWork:
		s.layoutWork(contentX, productionHeaderHeight, contentW, contentH)
	case pageSettings:
		s.layoutSettings(contentX, productionHeaderHeight, contentW, contentH)
	}
}

func (s *Shell) styleProductionControls() {
	ids := []int{idNavDashboard, idNavWork, idNavSettings, idTopNewTask, idTopNeedsYou, idTopReview}
	ids = append(ids, productionActionButtonIDs()...)
	for _, id := range ids {
		makeOwnerDrawButton(s.controls[id])
	}
}

func (s *Shell) handleProductionChromeCommand(id int) bool {
	switch id {
	case idNavDashboard:
		s.page = pageDashboard
		s.applyPageVisibility()
		s.refresh()
		s.layoutProduction()
		return true
	case idNavWork:
		s.page = pageWork
		s.jumpToNeedsAttention()
		s.applyPageVisibility()
		s.refresh()
		s.layoutProduction()
		return true
	case idNavSettings:
		s.page = pageSettings
		s.applyPageVisibility()
		s.refreshSettings(BuildSnapshot(s.eng, s.selectedTaskID))
		s.layoutProduction()
		return true
	case idTopNewTask:
		s.page = pageWork
		s.applyPageVisibility()
		s.refresh()
		s.layoutProduction()
		focusWindow(s.controls[idIntent])
		return true
	case idTopNeedsYou:
		if !s.jumpToNeedsAttention() {
			messageBox(s.hwnd, "Nothing needs you", "Workbench has no task waiting for a human decision.", mbOK|mbIconInformation)
			return true
		}
		s.page = pageWork
		s.applyPageVisibility()
		s.refresh()
		s.layoutProduction()
		return true
	case idTopReview:
		if !s.jumpToLatestReview() {
			messageBox(s.hwnd, "No review waiting", "There is no completed Workbench review artifact waiting to be opened or delivered.", mbOK|mbIconInformation)
			return true
		}
		s.page = pageWork
		s.applyPageVisibility()
		s.refresh()
		s.layoutProduction()
		return true
	}
	return false
}
