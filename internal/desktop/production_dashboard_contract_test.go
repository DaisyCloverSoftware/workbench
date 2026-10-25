package desktop

import (
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

// Inspect the owned startup body rather than gofmt's column padding. Comments
// and unrelated functions cannot establish the required initial dashboard.
func productionStartupUsesDashboard(source []byte) bool {
	file, err := parser.ParseFile(token.NewFileSet(), "startup.go", source, 0)
	if err != nil {
		return false
	}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "RunOwned" || fn.Body == nil || len(fn.Body.List) == 0 {
			continue
		}
		last, ok := fn.Body.List[len(fn.Body.List)-1].(*ast.ReturnStmt)
		if !ok || len(last.Results) != 1 {
			return false
		}
		call, ok := last.Results[0].(*ast.CallExpr)
		if !ok || len(call.Args) != 1 {
			return false
		}
		callee, ok := call.Fun.(*ast.Ident)
		if !ok || callee.Name != "runProductionShell" {
			return false
		}
		argument, ok := call.Args[0].(*ast.Ident)
		if !ok {
			return false
		}
		bound := false
		for _, stmt := range fn.Body.List[:len(fn.Body.List)-1] {
			assignment, ok := stmt.(*ast.AssignStmt)
			if !ok {
				continue
			}
			for i, lhs := range assignment.Lhs {
				name, ok := lhs.(*ast.Ident)
				if !ok || name.Name != argument.Name {
					continue
				}
				bound = false
				if len(assignment.Rhs) != len(assignment.Lhs) {
					continue
				}
				address, ok := assignment.Rhs[i].(*ast.UnaryExpr)
				if !ok || address.Op != token.AND {
					continue
				}
				literal, ok := address.X.(*ast.CompositeLit)
				if !ok {
					continue
				}
				kind, ok := literal.Type.(*ast.Ident)
				if !ok || kind.Name != "Shell" {
					continue
				}
				for _, elt := range literal.Elts {
					field, ok := elt.(*ast.KeyValueExpr)
					if !ok {
						continue
					}
					key, ok := field.Key.(*ast.Ident)
					if !ok || key.Name != "page" {
						continue
					}
					value, ok := field.Value.(*ast.Ident)
					bound = ok && value.Name == "pageDashboard"
				}
			}
		}
		return bound
	}
	return false
}

func TestProductionDesktopStartsInRealDashboardShell(t *testing.T) {
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve desktop source directory")
	}
	dir := filepath.Dir(here)
	startup, err := os.ReadFile(filepath.Join(dir, "startup_owned_windows.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !productionStartupUsesDashboard(startup) {
		t.Fatal("owned startup must pass a Shell initialized to pageDashboard to runProductionShell")
	}

	dashboard, err := os.ReadFile(filepath.Join(dir, "dashboard_windows.go"))
	if err != nil {
		t.Fatal(err)
	}
	dashboardText := string(dashboard)
	for _, want := range []string{"Recent activity", "Active tasks", "System status", "ChatGPT brain", "Autonomous worker health", "BuildDashboardSnapshot(s.eng)", "+ Delegate Task", "Review & Publish"} {
		if !strings.Contains(dashboardText, want) {
			t.Fatalf("production dashboard contract missing %q", want)
		}
	}
	for _, forbidden := range []string{"96.3%", "98 / 100", "fake worker", "synthetic health"} {
		if strings.Contains(strings.ToLower(dashboardText), strings.ToLower(forbidden)) {
			t.Fatalf("dashboard embedded invented operational telemetry %q", forbidden)
		}
	}
}

func TestProductionDashboardStartupShape(t *testing.T) {
	good := "package desktop\nfunc RunOwned() error { shell := &Shell{page:     pageDashboard}; return runProductionShell(shell) }"
	cases := []struct {
		name, source string
		want         bool
	}{
		{"original_spacing", good, true},
		{"different_spacing", strings.Replace(good, "page:     ", "page:\t", 1), true},
		{"wrong_page", strings.ReplaceAll(good, "pageDashboard", "pageSettings"), false},
		{"missing_page", strings.Replace(good, "page:     pageDashboard", "", 1), false},
		{"comment_is_not_wiring", strings.Replace(good, "page:     pageDashboard", "/*page: pageDashboard*/", 1), false},
		{"wrong_entry", strings.Replace(good, "runProductionShell", "runLegacyShell", 1), false},
		{"wrong_shell", strings.Replace(good, "runProductionShell(shell)", "runProductionShell(other)", 1), false},
		{"reassigned_shell", strings.Replace(good, "; return", "; shell = &Shell{page: pageSettings}; return", 1), false},
		{"unrelated_function", strings.Replace(good, "RunOwned", "OtherStartup", 1), false},
		{"invalid_source", "package desktop; func", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := productionStartupUsesDashboard([]byte(tc.source)); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			if formatted, err := format.Source([]byte(tc.source)); err == nil {
				if got := productionStartupUsesDashboard(formatted); got != tc.want {
					t.Fatalf("gofmt changed result: got %v, want %v", got, tc.want)
				}
			}
		})
	}
}

func TestProductionDashboardClipsParentPaintingAroundNativeOperationsControls(t *testing.T) {
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve desktop source directory")
	}
	dir := filepath.Dir(here)
	shell, err := os.ReadFile(filepath.Join(dir, "production_shell_windows.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(shell), "wsOverlappedWindow|wsVisible|wsClipChildren") {
		t.Fatal("production parent must use WS_CLIPCHILDREN so painted dashboard refreshes cannot cover native Operations controls")
	}
	style, err := os.ReadFile(filepath.Join(dir, "win32_clip_children_windows.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(style), "wsClipChildren = 0x02000000") {
		t.Fatal("WS_CLIPCHILDREN constant missing or incorrect")
	}
}
