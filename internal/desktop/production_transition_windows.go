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

// transitionProductionPage must remain a bounded native page swap. Data
// materialisation is posted back to the message queue so BM_CLICK can return to
// Windows before provider/project/list refresh work begins. That prevents page
// navigation itself from being classified as hung and gives the message pump an
// explicit scheduling boundary between user input and page refresh.
func (s *Shell) transitionProductionPage(page shellPage) {
	if s.hwnd != 0 {
		procSendMessageW.Call(s.hwnd, wmSetRedraw, 0, 0)
	}
	s.page = page
	s.applyProductionPageVisibility()
	if s.hwnd != 0 {
		procSendMessageW.Call(s.hwnd, wmSetRedraw, 1, 0)
		procPostMessageW.Call(s.hwnd, wmAppRefresh, 0, 0)
	}
}
