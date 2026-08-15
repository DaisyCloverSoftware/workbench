//go:build windows

package desktop

import "unsafe"

func (s *Shell) paintProductionWorkSettingsPanels() {
	if s.hwnd == 0 || s.page == pageDashboard {
		return
	}
	hdc, _, _ := user32.NewProc("GetDC").Call(s.hwnd)
	if hdc == 0 {
		return
	}
	defer user32.NewProc("ReleaseDC").Call(s.hwnd, hdc)

	var r nativeRect
	procGetClientRect.Call(s.hwnd, uintptr(unsafe.Pointer(&r)))
	width := int(r.Right-r.Left)
	height := int(r.Bottom-r.Top)
	if width < 1360 {
		width = 1360
	}
	pad := 16
	x := productionSidebarWidth + pad
	y := productionHeaderHeight + 8
	contentW := width - x - pad
	contentH := height - productionHeaderHeight - pad - 8

	var controls []int
	switch s.page {
	case pageWork:
		left := 270
		right := 320
		gap := 16
		center := contentW - left - right - gap*2
		if center < 430 {
			center = 430
		}
		roundedPanel(hdc, rectWH(x-8, y-4, left+16, contentH+2), productionPalette.Panel, productionPalette.Border, 12)
		roundedPanel(hdc, rectWH(x+left+gap-8, y-4, center+16, contentH+2), productionPalette.Panel, productionPalette.Border, 12)
		roundedPanel(hdc, rectWH(x+left+gap+center+gap-8, y-4, right+16, contentH+2), productionPalette.Panel, productionPalette.Border, 12)
		controls = s.workControlIDs()
	case pageSettings:
		gap := 18
		left := (contentW - gap) / 2
		right := contentW - left - gap
		roundedPanel(hdc, rectWH(x-8, y-4, left+16, contentH+2), productionPalette.Panel, productionPalette.Border, 12)
		roundedPanel(hdc, rectWH(x+left+gap-8, y-4, right+16, contentH+2), productionPalette.Panel, productionPalette.Border, 12)
		controls = s.settingsControlIDs()
	}

	// Child windows remain the actual interactive controls. Invalidate only the
	// children after painting the parent card surfaces; never invalidate the
	// parent here, otherwise its normal background paint would erase the cards.
	for _, id := range controls {
		if hwnd := s.controls[id]; hwnd != 0 {
			procInvalidateRect.Call(hwnd, 0, 1)
			procUpdateWindow.Call(hwnd)
		}
	}
}
