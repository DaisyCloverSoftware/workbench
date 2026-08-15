//go:build windows

package desktop

import "unsafe"

const (
	wmGetMinMaxInfo = 0x0024
	minimumWindowWidth  = int32(1260)
	minimumWindowHeight = int32(880)
)

type nativeMinMaxInfo struct {
	Reserved     point
	MaxSize      point
	MaxPosition  point
	MinTrackSize point
	MaxTrackSize point
}

func enforceMinimumTrackSize(lParam uintptr) {
	if lParam == 0 {
		return
	}
	info := (*nativeMinMaxInfo)(unsafe.Pointer(lParam))
	if info.MinTrackSize.X < minimumWindowWidth {
		info.MinTrackSize.X = minimumWindowWidth
	}
	if info.MinTrackSize.Y < minimumWindowHeight {
		info.MinTrackSize.Y = minimumWindowHeight
	}
}
