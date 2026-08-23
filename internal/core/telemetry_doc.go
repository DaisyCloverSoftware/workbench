package core

// Structured harness adapters may optionally report live execution telemetry on
// stderr using the WORKBENCH_PROGRESS: JSON record contract documented by
// HarnessProgress. Workbench validates every record before it can update durable
// task state; adapters that cannot measure completion remain stage-based.
