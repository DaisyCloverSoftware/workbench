//go:build windows

package desktop

import (
	"syscall"
	"unsafe"
)

const (
	lbsOwnerDrawFixed = 0x0010
	lbsHasStrings     = 0x0040
	lbGetText         = 0x0189
	lbGetTextLen      = 0x018A
	lbSetItemHeight   = 0x01A0
	odtListBox        = 2
)

func isOperationsOwnerDrawListID(id int) bool {
	switch id {
	case idOpsServerList, idOpsCIList, idOpsWindowsList, idOpsAIList,
		idOpsWaitingList, idOpsNeedsList, idOpsFullList,
		idOpsWorkersList, idOpsProjectsList, idOpsRecentList:
		return true
	default:
		return false
	}
}

func operationsOwnerDrawListNotifies(id int) bool {
	switch id {
	case idOpsServerList, idOpsCIList, idOpsWindowsList, idOpsAIList,
		idOpsWaitingList, idOpsNeedsList, idOpsFullList, idOpsRecentList:
		return true
	default:
		return false
	}
}

func recreateOperationsOwnerDrawListbox(s *Shell, id int) {
	if s == nil || s.hwnd == 0 || !isOperationsOwnerDrawListID(id) {
		return
	}
	if old := s.controls[id]; old != 0 {
		user32.NewProc("DestroyWindow").Call(old)
	}
	style := uintptr(wsChild | wsVisible | wsBorder | wsVScroll | lbsOwnerDrawFixed | lbsHasStrings)
	if operationsOwnerDrawListNotifies(id) {
		style |= lbsNotify
	}
	hwnd := s.control(id, "LISTBOX", "", style)
	if hwnd == 0 {
		return
	}
	if productionUIFont != 0 {
		procSendMessageW.Call(hwnd, wmSetFont, productionUIFont, 1)
	}
	applyDarkExplorerTheme(hwnd)
	procSendMessageW.Call(hwnd, lbSetItemHeight, 0, 22)
}

func (s *Shell) drawOperationsListItem(lParam uintptr) uintptr {
	if lParam == 0 {
		return 0
	}
	draw := (*drawItemStruct)(unsafe.Pointer(lParam))
	if draw.CtlType != odtListBox || !isOperationsOwnerDrawListID(int(draw.CtlID)) {
		return 0
	}
	background := productionPalette.PanelAlt
	if draw.ItemState&odsSelected != 0 {
		background = productionPalette.AccentSoft
	}
	fillRectColor(draw.HDC, draw.RCItem, background)
	if draw.ItemID == ^uint32(0) {
		return 1
	}

	length, _, _ := procSendMessageW.Call(draw.HwndItem, lbGetTextLen, uintptr(draw.ItemID), 0)
	if int32(length) < 0 {
		return 1
	}
	buf := make([]uint16, int(length)+1)
	if len(buf) == 0 {
		return 1
	}
	procSendMessageW.Call(draw.HwndItem, lbGetText, uintptr(draw.ItemID), uintptr(unsafe.Pointer(&buf[0])))
	text := syscall.UTF16ToString(buf)
	textRect := insetRect(draw.RCItem, 6, 1)
	drawTextStyled(draw.HDC, text, textRect, productionPalette.Text, 13, fwNormal, dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
	return 1
}
