//go:build windows

package desktop

const (
	wmSetRedraw   = 0x000B
	swpShowWindow = 0x0040
	swpHideWindow = 0x0080
)

var (
	procBeginDeferWindowPos = user32.NewProc("BeginDeferWindowPos")
	procDeferWindowPos      = user32.NewProc("DeferWindowPos")
	procEndDeferWindowPos   = user32.NewProc("EndDeferWindowPos")
	procIsWindowVisible     = user32.NewProc("IsWindowVisible")
)

type productionVisibilityTarget struct {
	hwnd    uintptr
	visible bool
}

// applyProductionPageVisibility batches the entire page swap into one Win32
// window-position transaction. The previous implementation called ShowWindow
// separately for every Work and Settings child control, which could synchronously
// cascade dozens of layout/paint messages inside the navigation button click.
func (s *Shell) applyProductionPageVisibility() {
	var targets []productionVisibilityTarget
	for _, id := range s.workControlIDs() {
		targets = append(targets, productionVisibilityTarget{hwnd: s.controls[id], visible: s.page == pageWork})
	}
	for _, id := range s.settingsControlIDs() {
		targets = append(targets, productionVisibilityTarget{hwnd: s.controls[id], visible: s.page == pageSettings})
	}

	changed := targets[:0]
	for _, target := range targets {
		if target.hwnd == 0 {
			continue
		}
		shown, _, _ := procIsWindowVisible.Call(target.hwnd)
		if (shown != 0) == target.visible {
			continue
		}
		changed = append(changed, target)
	}
	if len(changed) == 0 {
		return
	}

	hdwp, _, _ := procBeginDeferWindowPos.Call(uintptr(len(changed)))
	if hdwp == 0 {
		for _, target := range changed {
			showWindow(target.hwnd, target.visible)
		}
		return
	}
	for _, target := range changed {
		flags := uintptr(swpNoSize | swpNoMove | swpNoZOrder | swpNoActivate)
		if target.visible {
			flags |= swpShowWindow
		} else {
			flags |= swpHideWindow
		}
		next, _, _ := procDeferWindowPos.Call(hdwp, target.hwnd, 0, 0, 0, 0, 0, flags)
		if next == 0 {
			procEndDeferWindowPos.Call(hdwp)
			for _, fallback := range changed {
				showWindow(fallback.hwnd, fallback.visible)
			}
			return
		}
		hdwp = next
	}
	procEndDeferWindowPos.Call(hdwp)
}

// transitionProductionPage keeps the parent from repainting halfway through a
// page swap. Geometry is pre-laid out by layoutProduction on create/resize, so a
// navigation click only changes visibility and materialises the selected page's
// data before returning to the normal message loop.
func (s *Shell) transitionProductionPage(page shellPage) {
	if s.hwnd != 0 {
		procSendMessageW.Call(s.hwnd, wmSetRedraw, 0, 0)
	}
	s.page = page
	s.applyProductionPageVisibility()
	s.refreshProductionPage()
	if s.hwnd != 0 {
		procSendMessageW.Call(s.hwnd, wmSetRedraw, 1, 0)
	}
}
