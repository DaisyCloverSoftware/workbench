package core

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func unrealSmokeHasStreamingOutcomeWiring(src []byte) bool {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "smoke.go", src, 0)
	if err != nil {
		return false
	}
	want := []string{
		"stdout := &unrealSmokeCapture{}",
		"stderr := &unrealSmokeCapture{}",
		"cmd.Stdout = stdout",
		"cmd.Stderr = stderr",
		"runErr := cmd.Run()",
		"evidence := combineUnrealSmokeEvidence(stdout.finish(), stderr.finish())",
		"return unrealSmokeOutcome(runErr, probeCtx.Err(), evidence, version)",
	}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil || fn.Name.Name != "runUnrealSmoke" || fn.Body == nil || len(fn.Body.List) < len(want) {
			continue
		}
		statements := fn.Body.List[len(fn.Body.List)-len(want):]
		for i, statement := range statements {
			var text bytes.Buffer
			if err := format.Node(&text, fset, statement); err != nil || text.String() != want[i] {
				return false
			}
		}
		return true
	}
	return false
}

func TestUnrealSmokeProcessWiringUsesStreamingOutcome(t *testing.T) {
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate Windows source")
	}
	src, err := os.ReadFile(filepath.Join(filepath.Dir(here), "host_bridge_unreal_windows.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !unrealSmokeHasStreamingOutcomeWiring(src) {
		t.Fatal("Windows smoke must collect both streams, wait, finish records, then return the actual process outcome")
	}
}

func TestUnrealSmokeProcessWiringRejectsDisconnectedOrPrematureDiagnostics(t *testing.T) {
	valid := "package core\nfunc runUnrealSmoke() {\n" +
		"stdout := &unrealSmokeCapture{}\nstderr := &unrealSmokeCapture{}\n" +
		"cmd.Stdout = stdout\ncmd.Stderr = stderr\nrunErr := cmd.Run()\n" +
		"evidence := combineUnrealSmokeEvidence(stdout.finish(), stderr.finish())\n" +
		"return unrealSmokeOutcome(runErr, probeCtx.Err(), evidence, version)\n}\n"
	if !unrealSmokeHasStreamingOutcomeWiring([]byte(valid)) {
		t.Fatal("valid wiring rejected")
	}
	for _, bad := range []string{
		strings.Replace(valid, "cmd.Stderr = stderr", "cmd.Stderr = stdout", 1),
		strings.Replace(valid, "runErr := cmd.Run()", "runErr := pretendSuccess()", 1),
		strings.Replace(valid, "stdout.finish(), stderr.finish()", "stderr.finish(), stderr.finish()", 1),
		strings.Replace(valid, "unrealSmokeOutcome(runErr,", "unrealSmokeOutcome(nil,", 1),
		strings.Replace(valid, "cmd.Stdout = stdout", "// cmd.Stdout = stdout", 1),
		strings.Replace(valid, "runUnrealSmoke()", "unrelatedFunction()", 1),
		strings.Replace(valid, "runErr := cmd.Run()\nevidence :=", "evidence :=", 1),
	} {
		if unrealSmokeHasStreamingOutcomeWiring([]byte(bad)) {
			t.Fatal("disconnected/incorrect process wiring accepted")
		}
	}
}
