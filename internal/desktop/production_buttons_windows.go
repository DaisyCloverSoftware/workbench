//go:build windows

package desktop

import "unsafe"

func productionActionButtonIDs() []int {
	return []int{
		idAddProject, idRenameProject, idPinProject, idRemoveProject,
		idDelegate, idArchiveTask, idCancelTask, idResumeTask, idOpenReview, idRetryReview, idCopyBranch, idSaveNotes,
		idConnectProvider, idRescanProviders, idCopyMCP, idCopyChatGPTBootstrap, idSaveRouting,
		idSaveReviewPolicy, idSaveSecret, idRunUpdater,
	}
}

func isProductionActionButton(id int) bool {
	for _, candidate := range productionActionButtonIDs() {
		if id == candidate {
			return true
		}
	}
	return false
}

func (s *Shell) drawGenericProductionButton(lParam uintptr) uintptr {
	if lParam == 0 {
		return 0
	}
	item := (*drawItemStruct)(unsafe.Pointer(lParam))
	id := int(item.CtlID)
	if !isProductionActionButton(id) {
		return 0
	}

	fill := productionPalette.PanelAlt
	border := productionPalette.Border
	text := productionPalette.Text
	if productionPrimaryButton(id) {
		fill = productionPalette.AccentSoft
		border = productionPalette.Accent
	}
	if productionDangerButton(id) {
		fill = rgb(63, 28, 32)
		border = productionPalette.Red
		text = rgb(255, 199, 204)
	}
	if id == idOpenReview || id == idRetryReview {
		fill = rgb(24, 53, 46)
		border = productionPalette.Green
	}
	if item.ItemState&odsSelected != 0 {
		fill = border
	}
	if item.ItemState&odsDisabled != 0 {
		fill = productionPalette.Panel
		border = productionPalette.Border
		text = productionPalette.Muted
	}
	roundedPanel(item.HDC, item.RCItem, fill, border, 9)
	drawTextStyled(item.HDC, windowText(item.HwndItem), insetRect(item.RCItem, 8, 0), text, 11, fwSemiBold, dtCenter|dtSingleLine|dtVCenter|dtEndEllipsis)
	return 1
}

func productionPrimaryButton(id int) bool {
	switch id {
	case idAddProject, idDelegate, idResumeTask, idSaveNotes, idConnectProvider, idCopyMCP, idCopyChatGPTBootstrap, idSaveRouting, idSaveReviewPolicy, idSaveSecret, idRunUpdater:
		return true
	default:
		return false
	}
}

func productionDangerButton(id int) bool {
	return id == idRemoveProject || id == idCancelTask
}
