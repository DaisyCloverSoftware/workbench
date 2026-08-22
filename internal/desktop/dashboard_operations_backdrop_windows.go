//go:build windows

package desktop

const idOpsSurfaceBackdrop = 3625

// The production Dashboard originally painted a read-only Operations board
// directly into the parent window. Sprint 1 added native interactive controls
// over that area. This opaque sibling gives the interactive surface ownership
// of the lower dashboard without redesigning the parent painter; it prevents the
// retired board/resources text from bleeding through gaps between controls.
func (s *Shell) createOperationsDashboardBackdrop() {
	hwnd := s.static(idOpsSurfaceBackdrop, "")
	showWindow(hwnd, false)
	if hwnd != 0 {
		procSetWindowPos.Call(hwnd, 1, 0, 0, 0, 0, swpNoSize|swpNoMove|swpNoActivate)
	}
}

func (s *Shell) layoutOperationsDashboardBackdrop(clientWidth, clientHeight int) {
	hwnd := s.controls[idOpsSurfaceBackdrop]
	if hwnd == 0 {
		return
	}
	if s.page != pageDashboard {
		showWindow(hwnd, false)
		return
	}
	contentX := productionSidebarWidth + 20
	contentY := productionHeaderHeight + 18
	contentW := clientWidth - contentX - 18
	contentH := clientHeight - contentY - 18
	if contentW < 850 || contentH < 620 {
		showWindow(hwnd, false)
		return
	}
	metricsY := contentY + 68
	boardY := metricsY + 72 + 12
	bottom := contentY + contentH
	showWindow(hwnd, true)
	moveWindow(hwnd, contentX, boardY, contentW, bottom-boardY)
	procSetWindowPos.Call(hwnd, 1, 0, 0, 0, 0, swpNoSize|swpNoMove|swpNoActivate)

	// ShowWindow on an opaque STATIC can disturb sibling z-order on hosted
	// Windows runners even after placing the backdrop at HWND_BOTTOM. Reassert
	// every interactive Operations child above the backdrop before layout shows
	// them. Hidden child windows keep this z-order when they are shown moments
	// later, so PrintWindow and the real desktop both see the same usable surface.
	for _, id := range operationsDashboardControlIDs() {
		if child := s.controls[id]; child != 0 {
			procSetWindowPos.Call(child, 0, 0, 0, 0, 0, swpNoSize|swpNoMove|swpNoActivate)
		}
	}
}
