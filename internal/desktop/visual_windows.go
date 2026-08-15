//go:build windows

package desktop

import (
	"strings"
	"syscall"
	"unsafe"
)

const (
	wmPaint      = 0x000F
	wmEraseBkgnd = 0x0014
	wmDrawItem   = 0x002B
	bsOwnerDraw  = 0x0000000B
	odsSelected  = 0x0001
	odsDisabled  = 0x0004
	psSolid      = 0
	transparent  = 1
	dtLeft       = 0x0000
	dtCenter     = 0x0001
	dtRight      = 0x0002
	dtVCenter    = 0x0004
	dtWordBreak  = 0x0010
	dtSingleLine = 0x0020
	dtNoPrefix   = 0x0800
	dtEndEllipsis = 0x8000
	fwNormal     = 400
	fwSemiBold   = 600
	fwBold       = 700
)

var (
	procBeginPaint       = user32.NewProc("BeginPaint")
	procEndPaint         = user32.NewProc("EndPaint")
	procInvalidateRect   = user32.NewProc("InvalidateRect")
	procDrawTextW        = user32.NewProc("DrawTextW")
	procFillRect         = user32.NewProc("FillRect")
	procRoundRect        = gdi32.NewProc("RoundRect")
	procCreatePen        = gdi32.NewProc("CreatePen")
	procDeleteObject     = gdi32.NewProc("DeleteObject")
	procSelectObject     = gdi32.NewProc("SelectObject")
	procSetBkMode        = gdi32.NewProc("SetBkMode")
	procCreateFontW      = gdi32.NewProc("CreateFontW")
	procSetFocus         = user32.NewProc("SetFocus")
)

type paintStruct struct {
	HDC         uintptr
	Erase       int32
	RC          nativeRect
	Restore     int32
	IncUpdate   int32
	RGBReserved [32]byte
}

type drawItemStruct struct {
	CtlType    uint32
	CtlID      uint32
	ItemID     uint32
	ItemAction uint32
	ItemState  uint32
	HwndItem   uintptr
	HDC        uintptr
	RCItem     nativeRect
	ItemData   uintptr
}

var productionPalette = struct {
	Background uint32
	Sidebar    uint32
	Header     uint32
	Panel      uint32
	PanelAlt   uint32
	Border     uint32
	Text       uint32
	Muted      uint32
	Accent     uint32
	AccentSoft uint32
	Green      uint32
	Amber      uint32
	Red        uint32
	Purple     uint32
	Teal       uint32
}{
	Background: rgb(13, 18, 26),
	Sidebar:    rgb(15, 22, 31),
	Header:     rgb(16, 23, 33),
	Panel:      rgb(23, 31, 42),
	PanelAlt:   rgb(27, 36, 49),
	Border:     rgb(43, 55, 72),
	Text:       rgb(239, 243, 249),
	Muted:      rgb(145, 158, 176),
	Accent:     rgb(59, 130, 246),
	AccentSoft: rgb(28, 58, 99),
	Green:      rgb(34, 197, 94),
	Amber:      rgb(245, 158, 11),
	Red:        rgb(239, 68, 68),
	Purple:     rgb(168, 85, 247),
	Teal:       rgb(20, 184, 166),
}

func invalidateWindow(hwnd uintptr) {
	if hwnd != 0 {
		procInvalidateRect.Call(hwnd, 0, 0)
	}
}

func focusWindow(hwnd uintptr) {
	if hwnd != 0 {
		procSetFocus.Call(hwnd)
	}
}

func fillRectColor(hdc uintptr, rect nativeRect, color uint32) {
	brush, _, _ := procCreateSolidBrush.Call(uintptr(color))
	if brush == 0 {
		return
	}
	defer procDeleteObject.Call(brush)
	procFillRect.Call(hdc, uintptr(unsafe.Pointer(&rect)), brush)
}

func roundedPanel(hdc uintptr, rect nativeRect, fill, border uint32, radius int32) {
	brush, _, _ := procCreateSolidBrush.Call(uintptr(fill))
	pen, _, _ := procCreatePen.Call(psSolid, 1, uintptr(border))
	if brush == 0 || pen == 0 {
		if brush != 0 {
			procDeleteObject.Call(brush)
		}
		if pen != 0 {
			procDeleteObject.Call(pen)
		}
		return
	}
	oldBrush, _, _ := procSelectObject.Call(hdc, brush)
	oldPen, _, _ := procSelectObject.Call(hdc, pen)
	procRoundRect.Call(hdc, uintptr(rect.Left), uintptr(rect.Top), uintptr(rect.Right), uintptr(rect.Bottom), uintptr(radius), uintptr(radius))
	procSelectObject.Call(hdc, oldBrush)
	procSelectObject.Call(hdc, oldPen)
	procDeleteObject.Call(brush)
	procDeleteObject.Call(pen)
}

func drawTextStyled(hdc uintptr, text string, rect nativeRect, color uint32, size int32, weight int32, flags uint32) {
	if strings.TrimSpace(text) == "" {
		return
	}
	fontName := wstr("Segoe UI")
	font, _, _ := procCreateFontW.Call(
		uintptr(int64(-size)), 0, 0, 0, uintptr(weight), 0, 0, 0,
		1, 0, 0, 5, 0, uintptr(unsafe.Pointer(fontName)),
	)
	if font != 0 {
		oldFont, _, _ := procSelectObject.Call(hdc, font)
		defer func() {
			procSelectObject.Call(hdc, oldFont)
			procDeleteObject.Call(font)
		}()
	}
	procSetBkMode.Call(hdc, transparent)
	procSetTextColor.Call(hdc, uintptr(color))
	buf, _ := syscall.UTF16FromString(text)
	if len(buf) == 0 {
		return
	}
	procDrawTextW.Call(hdc, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)-1), uintptr(unsafe.Pointer(&rect)), uintptr(flags|dtNoPrefix))
}

func insetRect(rect nativeRect, x, y int32) nativeRect {
	return nativeRect{Left: rect.Left + x, Top: rect.Top + y, Right: rect.Right - x, Bottom: rect.Bottom - y}
}

func rectWH(x, y, width, height int) nativeRect {
	return nativeRect{Left: int32(x), Top: int32(y), Right: int32(x + width), Bottom: int32(y + height)}
}

func statusColor(status string) uint32 {
	s := strings.ToLower(strings.TrimSpace(status))
	switch {
	case strings.Contains(s, "need"), strings.Contains(s, "attention"):
		return productionPalette.Amber
	case strings.Contains(s, "fail"), strings.Contains(s, "error"), strings.Contains(s, "unavailable"):
		return productionPalette.Red
	case strings.Contains(s, "ready"), strings.Contains(s, "complete"), strings.Contains(s, "healthy"), strings.Contains(s, "online"), strings.Contains(s, "published"):
		return productionPalette.Green
	case strings.Contains(s, "waiting"), strings.Contains(s, "retry"):
		return productionPalette.Purple
	default:
		return productionPalette.Accent
	}
}
