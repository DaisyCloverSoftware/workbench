package core

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestUnrealSmokeZenDoesNotCombineUnrelatedLinesOrStreams(t *testing.T) {
	ok := "LogZenServiceInstance: Display: Unreal Zen Storage Server HTTP service status: OK!\n"
	unrelated := "LogWindows: Failed to load optional-profiler.dll (GetLastError=126)\n"
	for _, input := range []struct{ name, out, err string }{
		{"same stream", unrelated + ok, ""},
		{"separate streams", ok, unrelated},
		{"metadata substring", "LogCsvProfiler: Display: Metadata set : zenstreaming=0\n", unrelated},
	} {
		t.Run(input.name, func(t *testing.T) {
			if got := classifyUnrealSmokeFailure(input.out, input.err); got == "zen" {
				t.Fatal("unrelated failure was incorrectly attributed to Zen")
			}
		})
	}
}

func TestUnrealSmokeEvidencePreservesIndependentSignals(t *testing.T) {
	out := "LogZenServiceInstance: Display: Unreal Zen Storage Server HTTP service status: OK!\n" +
		"LogDerivedDataCache: Display: ZenLocal: Using ZenServer HTTP service. Status: OK!\n" +
		"LogDerivedDataCache: ZenShared: Disabled because Host is set to 'None'\n"
	e := capturedUnrealSmokeEvidence(out, "LogWindows: Failed to load optional-profiler.dll\n")
	if !e.zenServiceOK || !e.zenLocalOK || e.zenError || e.failureClass() != "nonzero-exit" {
		t.Fatalf("healthy local Zen was blamed: %s class=%s", e.summary(), e.failureClass())
	}
	e = combineUnrealSmokeEvidence(e, capturedUnrealSmokeEvidence("", "LogZenServiceInstance: Warning: Zen server connection failed\n"))
	if !e.zenServiceOK || !e.zenLocalOK || !e.zenError || e.failureClass() != "zen" {
		t.Fatal("positive and negative observations must both survive")
	}
}

func TestUnrealSmokeEvidenceGenuineFailuresAndPrecedence(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"Zen server connection failed", "zen"},
		{"LogZenServiceInstance: Error: cannot initialize service", "zen"},
		{"LogDerivedDataCache: Warning: ZenLocal: Readiness check failed", "zen"},
		{"Fatal error: TNotNull<Thing>", "tnotnull-assertion"},
		{"Assertion failed: Ptr", "assertion"},
		{"Fatal error: startup", "fatal"},
		{"Failed to open descriptor file", "project-descriptor"},
		{"Missing global shader", "shader-initialization"},
		{"LogCore: Engine exit requested", "quit-observed"},
		{"", "nonzero-exit"},
	} {
		if got := classifyUnrealSmokeFailure(tc.input, ""); got != tc.want {
			t.Errorf("%q: got=%s want=%s", tc.input, got, tc.want)
		}
	}
	e := capturedUnrealSmokeEvidence("Zen server connection failed\nLogCore: Engine exit requested\n", "Fatal error: startup\n")
	if e.failureClass() != "fatal" || !e.zenError || !e.quitObserved {
		t.Fatalf("precedence/signals lost: %s %s", e.failureClass(), e.summary())
	}
}

func TestUnrealSmokeEvidenceIgnoresMetadataAndSharedConfiguration(t *testing.T) {
	for _, out := range []string{
		"LogCsvProfiler: Metadata set: zenstreaming=0\n",
		"LogDerivedDataCache: ZenShared: Disabled because Host is set to 'None'\n",
		"LogDerivedDataCache: Unable to find inner node ZenShared for hierarchy Root\n",
		"LogModuleManager: loaded ZenEditor module\n",
		"LogZenServiceInstance: Display: version cache at test/error.cache\n",
	} {
		e := capturedUnrealSmokeEvidence(out, "LogWindows: Failed to load optional-profiler.dll\n")
		if e.zenError || e.failureClass() == "zen" {
			t.Fatalf("not a local Zen failure: %q", out)
		}
	}
	if e := capturedUnrealSmokeEvidence("Zen server connection", " failed"); e.zenError {
		t.Fatal("unterminated records from different streams were joined")
	}
}

func TestUnrealSmokeEvidenceProcessesLateRecordsAndAllChunkSizes(t *testing.T) {
	text := strings.Repeat("LogInit: ordinary startup progress\n", 1200) +
		"LogZenServiceInstance: Display: HTTP service status: OK!\r\n" +
		"LogDerivedDataCache: Display: ZenLocal: Status: OK!\n" +
		"LogInit: Display: Engine is initialized. Leaving FEngineLoop::Init()\n" +
		"LogRenderer: Video memory has been exhausted\n" +
		"LogCore: Engine exit requested"
	want := capturedUnrealSmokeEvidence(text, "")
	if !want.zenServiceOK || !want.zenLocalOK || !want.engineInitialized || !want.videoMemoryWarning || !want.quitObserved || want.oversizedLine {
		t.Fatalf("late evidence missing: %s", want.summary())
	}
	for _, chunk := range []int{1, 2, 7, 4096, 8192, len(text)} {
		c := &unrealSmokeCapture{}
		for at := 0; at < len(text); at += chunk {
			end := at + chunk
			if end > len(text) {
				end = len(text)
			}
			p := []byte(text[at:end])
			if n, err := c.Write(p); err != nil || n != len(p) {
				t.Fatalf("short write: %d %v", n, err)
			}
		}
		if got := c.finish(); got != want {
			t.Fatalf("chunk=%d changed evidence: %+v", chunk, got)
		}
		if got := c.finish(); got != want {
			t.Fatal("finish must be idempotent")
		}
		for _, b := range c.line {
			if b != 0 {
				t.Fatal("completed raw line retained")
			}
		}
	}
}

func TestUnrealSmokeEvidenceOversizedRecordsAreDiscardedAndReported(t *testing.T) {
	for _, newline := range []string{"", "\n"} {
		c := &unrealSmokeCapture{}
		_, _ = c.Write([]byte("Zen server connection failed " + strings.Repeat("x", unrealSmokeRecordLimit) + newline))
		e := c.finish()
		if !e.oversizedLine || e.zenError || e.failureClass() != "nonzero-exit" {
			t.Fatalf("oversized prefix classified: %+v", e)
		}
	}
	c := &unrealSmokeCapture{}
	_, _ = c.Write([]byte(strings.Repeat("x", unrealSmokeRecordLimit+1)))
	_, _ = c.Write([]byte("Zen server connection failed\nLogCore: Engine exit requested\n"))
	e := c.finish()
	if !e.oversizedLine || e.zenError || !e.quitObserved {
		t.Fatal(e.summary())
	}
	c = &unrealSmokeCapture{}
	_, _ = c.Write([]byte(strings.Repeat("x", unrealSmokeRecordLimit) + "\n"))
	if e := c.finish(); e.oversizedLine {
		t.Fatal("exact bound was incorrectly discarded")
	}
}

func TestUnrealSmokeEvidenceTimeoutStagesAndPrivacy(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"LogShaderCompilers: Display: Compiling shaders", "shader-work"},
		{"LogDerivedDataCache: Display: building derived data", "derived-data"},
		{"LogAssetRegistry: Display: asset registry scan", "asset-discovery"},
		{"LogInit: Display: Engine is initialized. Leaving FEngineLoop::Init()", "engine-ready"},
		{"LogInit: Display: startup continues", "initializing"},
		{"", "initializing"},
	} {
		if got := classifyUnrealSmokeTimeout(tc.input, ""); got != tc.want {
			t.Fatalf("got %s want %s", got, tc.want)
		}
	}
	marker := "private_test_payload_do_not_return"
	e := capturedUnrealSmokeEvidence("LogInit: "+marker+"\nLogRenderer: Video memory has been exhausted\n", "")
	if e.failureClass() != "nonzero-exit" || !e.videoMemoryWarning {
		t.Fatal("warning must not become process failure")
	}
	want := "diag=v3 zen_service_ok=false zen_local_ok=false zen_error=false engine_initialized=false quit_observed=false video_memory_warning=true oversized_line=false tail_stage=none records_after_tail_stage=-1 records_after_shader=-1"
	if got := e.summary(); got != want || strings.Contains(got, marker) {
		t.Fatal("summary must contain only fixed labels and bounded integers")
	}
}

func TestUnrealSmokeEvidenceReportsStageRecencyWithoutInventingGlobalStreamOrder(t *testing.T) {
	stdout := "LogDerivedDataCache: Display: ZenLocal: Status: OK!\n" +
		"LogShaderCompilers: Display: Compiling shaders\n" +
		"LogInit: later record one\nLogInit: later record two\n"
	stderr := "LogAssetRegistry: Display: asset registry scan\nLogInit: later stderr record\n"
	e := capturedUnrealSmokeEvidence(stdout, stderr)
	if e.timeoutClass() != "asset-discovery" || e.tailStage != "asset-discovery" || e.tailStageDistance != 1 {
		t.Fatalf("closest per-stream tail stage lost: %s class=%s", e.summary(), e.timeoutClass())
	}
	if !e.shaderTailKnown || e.shaderTailDistance != 2 {
		t.Fatalf("shader recency lost: %s", e.summary())
	}
	if !strings.Contains(e.summary(), "records_after_tail_stage=1") || !strings.Contains(e.summary(), "records_after_shader=2") {
		t.Fatalf("recency not reported: %s", e.summary())
	}
}

func TestUnrealSmokeEvidenceEngineReadyIsIndependentAndTailAware(t *testing.T) {
	e := capturedUnrealSmokeEvidence("LogAssetRegistry: asset registry scan\nLogInit: Display: Engine is initialized. Leaving FEngineLoop::Init()\nLogInit: later\n", "")
	if !e.engineInitialized || e.timeoutClass() != "engine-ready" || e.tailStage != "engine-ready" || e.tailStageDistance != 1 {
		t.Fatalf("engine-ready evidence lost: %s class=%s", e.summary(), e.timeoutClass())
	}
}

func TestUnrealSmokeEvidenceShaderClassRequiresProgressRecordNotCategoryNameAlone(t *testing.T) {
	e := capturedUnrealSmokeEvidence("LogShaderCompilers: Display: worker initialized\n", "")
	if e.shaderWork || e.timeoutClass() == "shader-work" || e.shaderTailKnown {
		t.Fatalf("shader category name alone became progress: %s", e.summary())
	}
	e = capturedUnrealSmokeEvidence("LogShaderCompilers: Display: Compiling shaders\n", "")
	if !e.shaderWork || e.timeoutClass() != "shader-work" || e.shaderTailDistance != 0 {
		t.Fatalf("actual shader progress record missed: %s", e.summary())
	}
}

func TestUnrealSmokeEvidenceNeverConvertsFailedProcessToSuccess(t *testing.T) {
	e := capturedUnrealSmokeEvidence("LogDerivedDataCache: ZenLocal: Status: OK!\nLogInit: Display: Engine is initialized.\nLogCore: Engine exit requested\n", "")
	privateError := errors.New("private_test_error_do_not_return")
	for _, contextErr := range []error{nil, context.DeadlineExceeded} {
		output, err := unrealSmokeOutcome(privateError, contextErr, e, "Unreal Engine 5.6.1")
		if output != "" || err == nil {
			t.Fatal("observations overrode failure")
		}
		if strings.Contains(err.Error(), privateError.Error()) {
			t.Fatal("raw process error escaped")
		}
		if !strings.Contains(err.Error(), "quit_observed=true") || !strings.Contains(err.Error(), "engine_initialized=true") || !strings.Contains(err.Error(), "zen_local_ok=true") {
			t.Fatal("independent observations not returned")
		}
		if contextErr != nil && !strings.HasPrefix(err.Error(), "Unreal headless smoke timed out: class=quit-observed ") {
			t.Fatal(err)
		}
		if contextErr == nil && !strings.Contains(err.Error(), "process_exit=-1") {
			t.Fatal(err)
		}
	}
}

func TestUnrealSmokeEvidencePreservesSuccessfulOutput(t *testing.T) {
	e := unrealSmokeEvidence{zenError: true, engineInitialized: true, videoMemoryWarning: true}
	output, err := unrealSmokeOutcome(nil, nil, e, "Unreal Engine 5.6.1")
	if err != nil || output != "Unreal headless smoke complete: Unreal Engine 5.6.1" {
		t.Fatalf("successful process result changed: %q %v", output, err)
	}
}
