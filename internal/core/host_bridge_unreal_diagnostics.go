package core

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

const unrealSmokeRecordLimit = 8 << 10

// Each process stream owns one collector. Finish is called only after Cmd.Run
// has joined the stream writers. No raw record is included in the result.
type unrealSmokeCapture struct {
	line     [unrealSmokeRecordLimit]byte
	used     int
	discard  bool
	evidence unrealSmokeEvidence
}

type unrealSmokeEvidence struct {
	failureRank        int
	zenServiceOK       bool
	zenLocalOK         bool
	zenError           bool
	quitObserved       bool
	videoMemoryWarning bool
	shaderWork         bool
	derivedData        bool
	assetDiscovery     bool
	oversizedLine      bool
	tailStage          string
	tailStageDistance  uint64
	tailStageKnown     bool
	shaderTailDistance uint64
	shaderTailKnown    bool
}

func (c *unrealSmokeCapture) Write(p []byte) (int, error) {
	for _, b := range p {
		if b == '\n' {
			c.finishLine()
			continue
		}
		if c.discard {
			continue
		}
		if c.used == len(c.line) {
			// Drop the entire overlong record, not just its tail. Otherwise a
			// clipped prefix could be mistaken for a complete diagnostic.
			clear(c.line[:c.used])
			c.used = 0
			c.discard = true
			c.evidence.oversizedLine = true
			continue
		}
		c.line[c.used] = b
		c.used++
	}
	return len(p), nil
}

func (c *unrealSmokeCapture) finishLine() {
	if c.discard || c.used != 0 {
		c.evidence.advanceRecord()
	}
	if !c.discard && c.used != 0 {
		c.evidence.observe(strings.ToLower(strings.TrimSpace(string(c.line[:c.used]))))
	}
	clear(c.line[:c.used])
	c.used = 0
	c.discard = false
}

func (c *unrealSmokeCapture) finish() unrealSmokeEvidence {
	c.finishLine()
	return c.evidence
}

func (e *unrealSmokeEvidence) advanceRecord() {
	if e.tailStageKnown {
		e.tailStageDistance++
	}
	if e.shaderTailKnown {
		e.shaderTailDistance++
	}
}

func (e *unrealSmokeEvidence) recordStage(stage string) {
	e.tailStage = stage
	e.tailStageDistance = 0
	e.tailStageKnown = true
}

func (e *unrealSmokeEvidence) recordShaderStage() {
	e.shaderWork = true
	e.shaderTailDistance = 0
	e.shaderTailKnown = true
	e.recordStage("shader")
}

func (e *unrealSmokeEvidence) recordFailure(rank int) {
	if e.failureRank == 0 || rank < e.failureRank {
		e.failureRank = rank
	}
}

func (e *unrealSmokeEvidence) observe(line string) {
	// Preserve the existing failure precedence, but never combine text from
	// unrelated records or streams to manufacture a subsystem error.
	switch {
	case strings.Contains(line, "tnotnull"):
		e.recordFailure(1)
	case strings.Contains(line, "assertion failed") || strings.Contains(line, "assert failed"):
		e.recordFailure(2)
	case strings.Contains(line, "fatal error") || strings.Contains(line, "app error called"):
		e.recordFailure(3)
	case strings.Contains(line, "failed to open descriptor file") || strings.Contains(line, "project file not found"):
		e.recordFailure(4)
	case strings.Contains(line, "missing global shader") || strings.Contains(line, "failed to compile global shader"):
		e.recordFailure(5)
	}

	zenService := strings.Contains(line, "logzenserviceinstance:")
	ddc := strings.Contains(line, "logderiveddatacache:")
	zenLocal := ddc && strings.Contains(line, "zenlocal:")
	zenRecord := zenService || zenLocal ||
		(ddc && strings.Contains(line, "zen server")) || strings.HasPrefix(line, "zen server ")
	// Optional ZenShared being unconfigured is not a local-service failure.
	failed := strings.Contains(line, ": error:") || strings.Contains(line, " failed") ||
		strings.HasPrefix(line, "failed ") || strings.Contains(line, "unable to ") ||
		strings.Contains(line, "timed out") || strings.Contains(line, "unhealthy")
	if zenRecord && failed {
		e.zenError = true
		e.recordFailure(6)
	}
	if zenService && strings.Contains(line, "http service") && strings.Contains(line, "status: ok") {
		e.zenServiceOK = true
		e.recordStage("zen-service")
	}
	if zenLocal && strings.Contains(line, "status: ok") {
		e.zenLocalOK = true
		e.recordStage("zen-local")
	}
	if strings.Contains(line, "engine exit requested") || strings.Contains(line, "requestengineexit") {
		e.quitObserved = true
		e.recordStage("quit")
	}
	if strings.Contains(line, "video memory has been exhausted") || strings.Contains(line, "out of video memory") {
		e.videoMemoryWarning = true
	}
	shaderProgress := strings.Contains(line, "compiling shaders") ||
		strings.Contains(line, "shaders left to compile") ||
		strings.Contains(line, "shader jobs") || strings.Contains(line, "shader compile jobs")
	if shaderProgress {
		e.recordShaderStage()
	}
	if ddc || strings.Contains(line, "derived data") || strings.Contains(line, "deriveddata") || strings.Contains(line, "ddc") {
		e.derivedData = true
		if !zenService && !zenLocal {
			e.recordStage("derived-data")
		}
	}
	if strings.Contains(line, "asset registry") || strings.Contains(line, "assetregistry") {
		e.assetDiscovery = true
		e.recordStage("asset-discovery")
	}
}

func combineUnrealSmokeEvidence(a, b unrealSmokeEvidence) unrealSmokeEvidence {
	if b.failureRank != 0 {
		a.recordFailure(b.failureRank)
	}
	a.zenServiceOK = a.zenServiceOK || b.zenServiceOK
	a.zenLocalOK = a.zenLocalOK || b.zenLocalOK
	a.zenError = a.zenError || b.zenError
	a.quitObserved = a.quitObserved || b.quitObserved
	a.videoMemoryWarning = a.videoMemoryWarning || b.videoMemoryWarning
	a.shaderWork = a.shaderWork || b.shaderWork
	a.derivedData = a.derivedData || b.derivedData
	a.assetDiscovery = a.assetDiscovery || b.assetDiscovery
	a.oversizedLine = a.oversizedLine || b.oversizedLine
	// Stdout/stderr are copied concurrently, so do not invent a global order.
	// Keep the recognised stage closest to the end of either individual stream.
	if b.tailStageKnown && (!a.tailStageKnown || b.tailStageDistance < a.tailStageDistance) {
		a.tailStage = b.tailStage
		a.tailStageDistance = b.tailStageDistance
		a.tailStageKnown = true
	}
	if b.shaderTailKnown && (!a.shaderTailKnown || b.shaderTailDistance < a.shaderTailDistance) {
		a.shaderTailDistance = b.shaderTailDistance
		a.shaderTailKnown = true
	}
	return a
}

func (e unrealSmokeEvidence) failureClass() string {
	switch e.failureRank {
	case 1:
		return "tnotnull-assertion"
	case 2:
		return "assertion"
	case 3:
		return "fatal"
	case 4:
		return "project-descriptor"
	case 5:
		return "shader-initialization"
	case 6:
		return "zen"
	}
	if e.quitObserved {
		return "quit-observed"
	}
	return "nonzero-exit"
}

func (e unrealSmokeEvidence) timeoutClass() string {
	if class := e.failureClass(); class != "nonzero-exit" {
		return class
	}
	switch e.tailStage {
	case "shader":
		return "shader-work"
	case "derived-data":
		return "derived-data"
	case "asset-discovery":
		return "asset-discovery"
	case "quit":
		return "quit-observed"
	default:
		return "initializing"
	}
}

func (e unrealSmokeEvidence) summary() string {
	stage := e.tailStage
	if !e.tailStageKnown {
		stage = "none"
	}
	stageDistance := int64(-1)
	if e.tailStageKnown {
		stageDistance = int64(e.tailStageDistance)
	}
	shaderDistance := int64(-1)
	if e.shaderTailKnown {
		shaderDistance = int64(e.shaderTailDistance)
	}
	return fmt.Sprintf("diag=v2 zen_service_ok=%t zen_local_ok=%t zen_error=%t quit_observed=%t video_memory_warning=%t oversized_line=%t tail_stage=%s records_after_tail_stage=%d records_after_shader=%d",
		e.zenServiceOK, e.zenLocalOK, e.zenError, e.quitObserved, e.videoMemoryWarning, e.oversizedLine,
		stage, stageDistance, shaderDistance)
}

// Keep the existing portable classifier entry points for callers and tests.
// The Windows smoke uses streaming collectors directly, retaining categorical
// signals even in the middle omitted by the old bounded head/tail capture.
func capturedUnrealSmokeEvidence(stdout, stderr string) unrealSmokeEvidence {
	a, b := &unrealSmokeCapture{}, &unrealSmokeCapture{}
	_, _ = a.Write([]byte(stdout))
	_, _ = b.Write([]byte(stderr))
	return combineUnrealSmokeEvidence(a.finish(), b.finish())
}

// Diagnostic signals never replace the actual process outcome. Return only
// fixed error text, a numeric process exit code and categorical observations.
func unrealSmokeOutcome(runErr, contextErr error, evidence unrealSmokeEvidence, version string) (string, error) {
	if runErr == nil {
		return "Unreal headless smoke complete: " + version, nil
	}
	if contextErr != nil {
		return "", fmt.Errorf("Unreal headless smoke timed out: class=%s %s", evidence.timeoutClass(), evidence.summary())
	}
	exitCode := -1
	var exitErr *exec.ExitError
	if errors.As(runErr, &exitErr) {
		exitCode = exitErr.ExitCode()
	}
	return "", fmt.Errorf("Unreal headless smoke failed: class=%s process_exit=%d %s", evidence.failureClass(), exitCode, evidence.summary())
}
