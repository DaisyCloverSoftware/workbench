//go:build windows

package desktop

// WS_CLIPCHILDREN excludes child-window rectangles from parent painting. The
// Operations dashboard uses native child controls over a painted dashboard
// surface, so the production parent must not repaint over live telemetry rows.
const wsClipChildren = 0x02000000
