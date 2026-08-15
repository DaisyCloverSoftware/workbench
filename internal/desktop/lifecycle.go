package desktop

// CanRecoverInterruptedTasks decides whether startup has an exclusive-enough
// control-plane ownership proof to restart durable work. Normal Windows desktop
// startup owns a per-user named mutex before main runs. MCP listener ownership
// is an independent fallback/second line of defence if mutex creation was not
// available on an unusual host. Without either proof, Workbench may remain open
// for diagnosis/settings but must not automatically restart old coding work.
func CanRecoverInterruptedTasks(processOwnershipConfirmed, mcpListenerOwned bool) bool {
	return processOwnershipConfirmed || mcpListenerOwned
}
