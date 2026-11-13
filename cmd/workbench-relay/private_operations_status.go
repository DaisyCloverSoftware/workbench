package main

// This projection is populated only from core.RunOperationsScript's typed
// result, not a script payload or historical output. Existing payload filtering
// is unchanged. Losing a payload cannot discard executor-owned lifecycle facts
// or the independently schema-validated, forward-only disposition.
func markOperationPayloadUnavailable(out *privateControlOutbox, withheld bool) {
	if out.OperationStatus == nil {
		return
	}
	status := *out.OperationStatus
	status.EvidenceComplete = false
	if withheld {
		status.OutputWithheld = true
	} else {
		status.OutputTruncated = true
	}
	out.OperationStatus = &status
}
