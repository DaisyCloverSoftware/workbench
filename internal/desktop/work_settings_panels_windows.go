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
	clientW := int(r.Right - r.Left)
	clientH := int(r.Bottom - r.Top)
	x, y, contentW, contentH := productionContentGeometry(clientW, clientH)

	var controls []int
	switch s.page {
	case pageWork:
		left, center, right, gap := productionWorkColumns(contentW)
		roundedPanel(hdc, rectWH(x-6, y+4, left+12, contentH-8), productionPalette.Panel, productionPalette.Border, 12)
		roundedPanel(hdc, rectWH(x+left+gap-6, y+4, center+12, contentH-8), productionPalette.Panel, productionPalette.Border, 12)
		roundedPanel(hdc, rectWH(x+left+gap+center+gap-6, y+4, right+12, contentH-8), productionPalette.Panel, productionPalette.Border, 12)
		controls = s.workControlIDs()
	case pageSettings:
		gap := 16
		left := (contentW - gap) / 2
		right := contentW - left - gap
		roundedPanel(hdc, rectWH(x-6, y+4, left+12, contentH-8), productionPalette.Panel, productionPalette.Border, 12)
		roundedPanel(hdc, rectWH(x+left+gap-6, y+4, right+12, contentH-8), productionPalette.Panel, productionPalette.Border, 12)
		controls = s.settingsControlIDs()
	}

	// Child windows remain the actual interactive controls. Mark them dirty and
	// let the normal message loop repaint them after the parent WM_PAINT returns.
	// Calling UpdateWindow here re-enters child painting synchronously from inside
	// the parent paint path and can starve the desktop message pump on real Windows
	// machines even though a final frame is still visible.
	for _, id := range controls {
		if hwnd := s.controls[id]; hwnd != 0 {
			procInvalidateRect.Call(hwnd, 0, 1)
		}
	}
}
