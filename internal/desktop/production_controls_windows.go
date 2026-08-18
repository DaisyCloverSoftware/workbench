//go:build windows

package desktop

import (
	"syscall"
	"unsafe"
)

const (
	gwlStyle        = ^uintptr(15) // Win32 GWL_STYLE (-16)
	gwlExStyle      = ^uintptr(19) // Win32 GWL_EXSTYLE (-20)
	swpNoSize       = 0x0001
	swpNoMove       = 0x0002
	swpNoZOrder     = 0x0004
	swpNoActivate   = 0x0010
	swpFrameChanged = 0x0020
	emSetMargins    = 0x00D3
	ecLeftMargin    = 0x0001
	ecRightMargin   = 0x0002
)

var (
	productionPanelBrush   uintptr
	productionFieldBrush   uintptr
	productionSidebarBrush uintptr
	productionUIFont       uintptr
	uxtheme                 = syscall.NewLazyDLL("uxtheme.dll")
	procSetWindowTheme      = uxtheme.NewProc("SetWindowTheme")
	procSetWindowPos        = user32.NewProc("SetWindowPos")
)

func initProductionControlSurfaces() {
	if productionPanelBrush == 0 {
		productionPanelBrush, _, _ = procCreateSolidBrush.Call(uintptr(productionPalette.Panel))
	}
	if productionFieldBrush == 0 {
		productionFieldBrush, _, _ = procCreateSolidBrush.Call(uintptr(productionPalette.PanelAlt))
	}
	if productionSidebarBrush == 0 {
		productionSidebarBrush, _, _ = procCreateSolidBrush.Call(uintptr(productionPalette.Sidebar))
	}
	if productionUIFont == 0 {
		fontName := wstr("Segoe UI")
		fontHeight := int32(-15)
		productionUIFont, _, _ = procCreateFontW.Call(
			uintptr(int64(fontHeight)), 0, 0, 0, uintptr(fwNormal), 0, 0, 0,
			1, 0, 0, 5, 0, uintptr(unsafe.Pointer(fontName)),
		)
	}
}

func releaseProductionControlSurfaces() {
	for _, object := range []*uintptr{&productionPanelBrush, &productionFieldBrush, &productionSidebarBrush, &productionUIFont} {
		if *object != 0 {
			procDeleteObject.Call(*object)
			*object = 0
		}
	}
}

func (s *Shell) applyProductionControlTheme() {
	labels := map[int]string{
		idProjectsLabel:    "Projects",
		idIntentLabel:      "Autonomous delegation — use when ChatGPT decides a worker is needed",
		idTasksLabel:       "Delegated tasks",
		idReportLabel:      "Result & activity",
		idNotesLabel:       "Project context",
		idAttentionLabel:   "Human decision — only answer when Workbench genuinely needs you",
		idSettingsTitle:    "PRIMARY · This PC · ChatGPT Chat · normal Workbench brain",
		idProvidersLabel:   "Chat-first routing · primary brain + autonomous workers",
		idMCPLabel:         "ChatGPT transport · local MCP + runner relay status",
		idRunnerLabel:      "Workbench Runner SSH host",
		idHarnessLabel:     "Structured harness adapter executable",
		idNotifyLabel:      "Human-interrupt command · {message}",
		idReviewLabel:      "Review delivery · active project",
		idVaultLabel:       "Encrypted vault · Windows DPAPI · hidden from AI",
		idMaintenanceLabel: "Maintenance",
	}
	for id, text := range labels {
		setWindowText(s.controls[id], text)
	}

	if productionUIFont != 0 {
		for _, hwnd := range s.controls {
			if hwnd != 0 {
				procSendMessageW.Call(hwnd, wmSetFont, productionUIFont, 1)
			}
		}
	}

	for _, id := range productionFieldControlIDs() {
		hwnd := s.controls[id]
		if hwnd == 0 {
			continue
		}
		stripProductionFieldChrome(hwnd)
		applyDarkExplorerTheme(hwnd)
		if isProductionEditControl(id) {
			margin := uintptr(7 | (7 << 16))
			procSendMessageW.Call(hwnd, emSetMargins, ecLeftMargin|ecRightMargin, margin)
		}
	}
	for _, id := range []int{idShowArchivedTasks, idProtectWork, idAllowMetered, idPublishReviews} {
		applyDarkExplorerTheme(s.controls[id])
	}
}

func productionFieldControlIDs() []int {
	return []int{
		idProjectList, idProjectName, idIntent, idTaskList, idReport, idAnswer, idNotes,
		idProviderList, idRunnerHost, idHarnessCommand, idNotifyCommand, idReviewRemote,
		idSecretName, idSecretValue, idSecretList,
	}
}

func isProductionEditControl(id int) bool {
	switch id {
	case idProjectName, idIntent, idReport, idAnswer, idNotes, idRunnerHost, idHarnessCommand,
		idNotifyCommand, idReviewRemote, idSecretName, idSecretValue:
		return true
	default:
		return false
	}
}

func stripProductionFieldChrome(hwnd uintptr) {
	if hwnd == 0 {
		return
	}
	style, _, _ := procGetWindowLongPtrW.Call(hwnd, gwlStyle)
	procSetWindowLongPtrW.Call(hwnd, gwlStyle, style&^wsBorder)
	exStyle, _, _ := procGetWindowLongPtrW.Call(hwnd, gwlExStyle)
	procSetWindowLongPtrW.Call(hwnd, gwlExStyle, exStyle&^wsExClientEdge)
	procSetWindowPos.Call(hwnd, 0, 0, 0, 0, swpNoSize|swpNoMove|swpNoZOrder|swpNoActivate|swpFrameChanged)
}

func applyDarkExplorerTheme(hwnd uintptr) {
	if hwnd == 0 {
		return
	}
	theme := wstr("DarkMode_Explorer")
	procSetWindowTheme.Call(hwnd, uintptr(unsafe.Pointer(theme)), 0)
}

func (s *Shell) productionControlColor(message uint32, wParam, lParam uintptr) uintptr {
	hdc := wParam
	if hdc == 0 {
		return 0
	}
	text := productionPalette.Text
	background := productionPalette.Panel
	brush := productionPanelBrush

	switch message {
	case wmCtlColorEdit, wmCtlColorListBox:
		background = productionPalette.PanelAlt
		brush = productionFieldBrush
	case wmCtlColorStatic:
		if lParam == s.controls[idBrand] {
			background = productionPalette.Sidebar
			brush = productionSidebarBrush
		}
		procSetBkMode.Call(hdc, transparent)
	case wmCtlColorBtn:
		procSetBkMode.Call(hdc, transparent)
	}
	if brush == 0 {
		brush = s.backgroundBrush
		background = productionPalette.Background
	}
	procSetTextColor.Call(hdc, uintptr(text))
	procSetBkColor.Call(hdc, uintptr(background))
	return brush
}
