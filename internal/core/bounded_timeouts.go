package core

import "time"

// MachineCommandTimeout returns the same bounded execution timeout used by
// InspectMachine and RunMachineCommand. Internal supervisors use it to grant a
// long-running command enough forward-progress lease without hard-coding a
// second timeout policy.
func MachineCommandTimeout(timeoutSeconds int) time.Duration {
	timeout := defaultMachineCommandTimeout
	if timeoutSeconds > 0 {
		timeout = time.Duration(timeoutSeconds) * time.Second
	}
	if timeout > maxMachineCommandTimeout {
		timeout = maxMachineCommandTimeout
	}
	if timeout < time.Second {
		timeout = time.Second
	}
	return timeout
}

// OperationsScriptTimeout returns the bounded script runtime used by
// RunOperationsScript. Source preparation and cleanup happen outside this
// runtime, so supervisors should add their own small grace period.
func OperationsScriptTimeout(timeoutSeconds int) time.Duration {
	timeout := defaultOperationsScriptTimeout
	if timeoutSeconds > 0 {
		timeout = time.Duration(timeoutSeconds) * time.Second
	}
	if timeout > maxOperationsScriptTimeout {
		timeout = maxOperationsScriptTimeout
	}
	if timeout < time.Second {
		timeout = time.Second
	}
	return timeout
}
