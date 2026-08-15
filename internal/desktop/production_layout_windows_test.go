//go:build windows

package desktop

import "testing"

func TestProductionContentGeometryFitsStandardDesktop(t *testing.T) {
	x, y, width, height := productionContentGeometry(1280, 840)
	if x != productionSidebarWidth+16 || y != productionHeaderHeight {
		t.Fatalf("content origin=(%d,%d) want (%d,%d)", x, y, productionSidebarWidth+16, productionHeaderHeight)
	}
	if width <= 900 || height <= 700 {
		t.Fatalf("content area=%dx%d unexpectedly cramped", width, height)
	}
	if x+width+16 > 1280 || y+height+16 > 840 {
		t.Fatalf("content geometry overflows 1280x840: origin=(%d,%d) size=%dx%d", x, y, width, height)
	}
}

func TestProductionWorkColumnsRemainInsideContentWidth(t *testing.T) {
	for _, width := range []int{760, 900, 1064, 1280} {
		left, center, right, gap := productionWorkColumns(width)
		if left <= 0 || center <= 0 || right <= 0 || gap <= 0 {
			t.Fatalf("width %d produced invalid columns %d/%d/%d gap %d", width, left, center, right, gap)
		}
		if left+center+right+2*gap != width {
			t.Fatalf("width %d: columns consume %d", width, left+center+right+2*gap)
		}
	}
}

func TestProductionMinimumWindowFitsCommon1360x900WorkArea(t *testing.T) {
	if minimumWindowWidth > 1360 || minimumWindowHeight > 900 {
		t.Fatalf("minimum production window %dx%d exceeds common 1360x900 work area", minimumWindowWidth, minimumWindowHeight)
	}
}
