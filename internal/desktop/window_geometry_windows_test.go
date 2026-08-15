//go:build windows

package desktop

import (
	"testing"
	"unsafe"
)

func TestEnforceMinimumTrackSizeRaisesOnlyUndersizedDimensions(t *testing.T) {
	info := nativeMinMaxInfo{MinTrackSize: point{X: 800, Y: 700}}
	enforceMinimumTrackSize(uintptr(unsafe.Pointer(&info)))
	if info.MinTrackSize.X != minimumWindowWidth || info.MinTrackSize.Y != minimumWindowHeight {
		t.Fatalf("minimum track size=%#v want %dx%d", info.MinTrackSize, minimumWindowWidth, minimumWindowHeight)
	}

	larger := nativeMinMaxInfo{MinTrackSize: point{X: 1600, Y: 1000}}
	enforceMinimumTrackSize(uintptr(unsafe.Pointer(&larger)))
	if larger.MinTrackSize.X != 1600 || larger.MinTrackSize.Y != 1000 {
		t.Fatalf("existing larger minimum was reduced: %#v", larger.MinTrackSize)
	}
}

func TestEnforceMinimumTrackSizeAcceptsNilMessagePayload(t *testing.T) {
	enforceMinimumTrackSize(0)
}
