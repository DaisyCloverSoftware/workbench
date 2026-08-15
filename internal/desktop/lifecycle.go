package desktop

// CanRecoverInterruptedTasks decides whether desktop startup has a genuine
// exclusivity proof before restarting durable work. The shipped Windows desktop
// owns a per-user named mutex before main runs. MCP listener acquisition is not
// an exclusivity proof because the desktop may intentionally fall forward to a
// different free port. If mutex ownership could not be established, Workbench
// may remain open for diagnosis/settings but must not automatically restart old
// coding work.
func CanRecoverInterruptedTasks(processOwnershipConfirmed bool) bool {
	return processOwnershipConfirmed
}
