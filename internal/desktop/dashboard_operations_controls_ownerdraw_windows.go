//go:build windows

package desktop

// createOperationsDashboardControlsOwnerDraw is the production Operations
// control creator. Owner-draw LISTBOX styles must be present in CreateWindowEx;
// setting them after creation does not reliably switch Win32 into WM_DRAWITEM
// mode. The existing dashboard command/state code continues to address these
// controls by the same IDs.
func (s *Shell) createOperationsDashboardControlsOwnerDraw() {
	interactiveListStyle := uintptr(wsChild | wsVisible | wsBorder | wsVScroll | lbsNotify | lbsOwnerDrawFixed | lbsHasStrings)
	readOnlyListStyle := uintptr(wsChild | wsVisible | wsBorder | wsVScroll | lbsOwnerDrawFixed | lbsHasStrings)

	for _, control := range operationsLaneControls {
		s.control(control.Header, "BUTTON", workLaneTitle(control.Lane), wsChild|wsVisible|wsTabStop|bsPushButton)
		hwnd := s.control(control.List, "LISTBOX", "", interactiveListStyle)
		prepareOperationsOwnerDrawList(hwnd)
	}

	s.control(idOpsFullHeader, "BUTTON", "All operations", wsChild|wsVisible|wsTabStop|bsPushButton)
	prepareOperationsOwnerDrawList(s.control(idOpsFullList, "LISTBOX", "", interactiveListStyle))
	s.control(idOpsDetails, "EDIT", "", wsChild|wsVisible|wsBorder|wsVScroll|esMultiline|esAutoVScroll|esReadOnly)
	applyDarkExplorerTheme(s.controls[idOpsDetails])
	s.control(idOpsPriorityUp, "BUTTON", "Priority ↑", wsChild|wsVisible|wsTabStop|bsPushButton)
	s.control(idOpsPriorityDown, "BUTTON", "Priority ↓", wsChild|wsVisible|wsTabStop|bsPushButton)
	s.control(idOpsOpenTask, "BUTTON", "Open task", wsChild|wsVisible|wsTabStop|bsPushButton)
	s.control(idOpsCloseDetails, "BUTTON", "Close details", wsChild|wsVisible|wsTabStop|bsPushButton)

	s.static(idOpsWorkersLabel, "Worker assignments")
	prepareOperationsOwnerDrawList(s.control(idOpsWorkersList, "LISTBOX", "", readOnlyListStyle))
	s.static(idOpsProjectsLabel, "Project activity")
	prepareOperationsOwnerDrawList(s.control(idOpsProjectsList, "LISTBOX", "", readOnlyListStyle))
	s.static(idOpsRecentLabel, "Recent outcomes")
	prepareOperationsOwnerDrawList(s.control(idOpsRecentList, "LISTBOX", "", interactiveListStyle))

	s.hideOperationsDashboardControls()
}

func prepareOperationsOwnerDrawList(hwnd uintptr) {
	if hwnd == 0 {
		return
	}
	applyDarkExplorerTheme(hwnd)
	procSendMessageW.Call(hwnd, lbSetItemHeight, 0, 22)
}
